package main

import (
	"C"
	"crypto/tls"
	"encoding/json"
	"log"
	"strconv"
	"unsafe"

	"crypto/x509"
	"fmt"
	"os"
	"sync"

	"github.com/fluent/fluent-bit-go/output"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ConnectionConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	VHost        string
	ExchangeName string
	ExchangeType string
	TLSEnabled   bool
	TLSConfig    *tls.Config
}

var (
	config                   *ConnectionConfig
	connection               *amqp.Connection
	channel                  *amqp.Channel
	exchangeName             string
	routingKey               string
	routingKeyDelimiter      string
	removeRkValuesFromRecord bool
	addTagToRecord           bool
	addTimestampToRecord     bool
	contentEncoding          string
	connectionMutex          sync.Mutex // Mutex to make connection initialization thread-safe
)

//export FLBPluginRegister
func FLBPluginRegister(def unsafe.Pointer) int {
	// Gets called only once when the plugin.so is loaded
	return output.FLBPluginRegister(def, "rabbitmq", "RabbitMQ output plugin")
}

//export FLBPluginInit
func FLBPluginInit(plugin unsafe.Pointer) int {
	// Gets called only once for each instance you have configured.
	var err error

	config := &ConnectionConfig{
		Host:         output.FLBPluginConfigKey(plugin, "RabbitHost"),
		Port:         output.FLBPluginConfigKey(plugin, "RabbitPort"),
		User:         output.FLBPluginConfigKey(plugin, "RabbitUser"),
		Password:     output.FLBPluginConfigKey(plugin, "RabbitPassword"),
		VHost:        output.FLBPluginConfigKey(plugin, "RabbitVHost"),
		ExchangeName: output.FLBPluginConfigKey(plugin, "ExchangeName"),
		ExchangeType: output.FLBPluginConfigKey(plugin, "ExchangeType"),
	}

	exchangeName = config.ExchangeName
	routingKey = output.FLBPluginConfigKey(plugin, "RoutingKey")
	routingKeyDelimiter = output.FLBPluginConfigKey(plugin, "RoutingKeyDelimiter")
	removeRkValuesFromRecordStr := output.FLBPluginConfigKey(plugin, "RemoveRkValuesFromRecord")
	addTagToRecordStr := output.FLBPluginConfigKey(plugin, "AddTagToRecord")
	addTimestampToRecordStr := output.FLBPluginConfigKey(plugin, "AddTimestampToRecord")
	contentEncoding = output.FLBPluginConfigKey(plugin, "ContentEncoding")

	tlsCertFile := output.FLBPluginConfigKey(plugin, "TLSCertFile")
	tlsKeyFile := output.FLBPluginConfigKey(plugin, "TLSKeyFile")
	tlsCACertFile := output.FLBPluginConfigKey(plugin, "TLSCACertFile")
	tlsInsecureSkipVerifyStr := output.FLBPluginConfigKey(plugin, "TLSInsecureSkipVerify")
	tlsEnabledStr := output.FLBPluginConfigKey(plugin, "TLSEnabled")
	if tlsEnabledStr == "" {
		logInfo("TLSEnabled not specified, defaulting to false.")
		tlsEnabledStr = "false" // Default value if not specified
	}

	if len(routingKeyDelimiter) < 1 {
		routingKeyDelimiter = "."
		logInfo("The routing-key-delimiter is set to the default value '" + routingKeyDelimiter + "' ")
	}

	removeRkValuesFromRecord, err = strconv.ParseBool(removeRkValuesFromRecordStr)
	if err != nil {
		logError("Couldn't parse RemoveRkValuesFromRecord to boolean: ", err)
		return output.FLB_ERROR
	}

	addTagToRecord, err = strconv.ParseBool(addTagToRecordStr)
	if err != nil {
		logError("Couldn't parse AddTagToRecord to boolean: ", err)
		return output.FLB_ERROR
	}

	addTimestampToRecord, err = strconv.ParseBool(addTimestampToRecordStr)
	if err != nil {
		logError("Couldn't parse AddTimestampToRecord to boolean: ", err)
		return output.FLB_ERROR
	}

	err = RoutingKeyIsValid(routingKey, routingKeyDelimiter)
	if err != nil {
		logError("The Parsing of the Routing-Key failed: ", err)
		return output.FLB_ERROR
	}

	if len(contentEncoding) < 1 {
		contentEncoding = ""
	}

	tlsEnabled, err := strconv.ParseBool(tlsEnabledStr)
	if err != nil {
		logError("Couldn't parse TLSEnabled to boolean: ", err)
		return output.FLB_ERROR
	}

	if tlsEnabled {
		config.TLSEnabled = true
		config.TLSConfig = &tls.Config{}

		if tlsCertFile != "" && tlsKeyFile != "" {
			cert, err := tls.LoadX509KeyPair(tlsCertFile, tlsKeyFile)
			if err != nil {
				logError("Failed to load TLS certificate and key: ", err)
				return output.FLB_ERROR
			}
			config.TLSConfig.Certificates = []tls.Certificate{cert}
		}

		if tlsCACertFile != "" {
			caCertPool, err := loadCACert(tlsCACertFile)
			if err != nil {
				logError("Failed to load TLS CA certificate: ", err)
				return output.FLB_ERROR
			}
			config.TLSConfig.RootCAs = caCertPool
		}

		if tlsInsecureSkipVerifyStr != "" {
			tlsInsecureSkipVerify, err := strconv.ParseBool(tlsInsecureSkipVerifyStr)
			if err != nil {
				logError("Couldn't parse TLSInsecureSkipVerify to boolean: ", err)
				return output.FLB_ERROR
			}
			config.TLSConfig.InsecureSkipVerify = tlsInsecureSkipVerify
		}
	}

	err = initConnection(config)
	if err != nil {
		return output.FLB_ERROR
	}

	err = channel.ExchangeDeclare(
		config.ExchangeName, // name
		config.ExchangeType, // type
		true,                // durable
		false,               // auto-deleted
		false,               // internal
		false,               // no-wait
		nil,                 // arguments
	)

	if err != nil {
		logError("Failed to declare an exchange: ", err)
		connection.Close()
		return output.FLB_ERROR
	}

	return output.FLB_OK
}

//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	// Gets called with a batch of records to be written to an instance.
	// Create Fluent Bit decoder
	dec := output.NewDecoder(data, int(length))

	// Iterate Records
	for {
		// Extract Record
		ret, ts, record := output.GetRecord(dec)
		if ret != 0 {
			break
		}

		timestamp := ts.(output.FLBTime)

		parsedRecord := ParseRecord(record)

		if addTagToRecord {
			parsedRecord["@tag"] = C.GoString(tag)
		}
		if addTimestampToRecord {
			parsedRecord["@timestamp"] = timestamp.String()
		}

		rk, err := CreateRoutingKey(routingKey, &parsedRecord, routingKeyDelimiter)
		if err != nil {
			logError("Couldn't create the Routing-Key", err)
			continue
		}

		jsonString, err := json.Marshal(parsedRecord)

		if err != nil {
			logError("Couldn't parse record: ", err)
			continue
		}

		err = channel.Publish(
			exchangeName, // exchange
			rk,           // routing key
			false,        // mandatory
			false,        // immediate
			amqp.Publishing{
				DeliveryMode:    amqp.Persistent,
				ContentType:     "application/json",
				ContentEncoding: contentEncoding,
				Body:            jsonString,
			})
		if err != nil {
			if err == amqp.ErrClosed {
				logError("Connection to RabbitMQ was closed, trying to reconnect... ", err)
				err = initConnection(config)
				if err != nil {
					return output.FLB_ERROR
				} else {
					return output.FLB_RETRY
				}
			}
			logError("Couldn't publish record: ", err)
			return output.FLB_ERROR
		}
	}
	return output.FLB_OK
}

//export FLBPluginExit
func FLBPluginExit() int {
	connectionMutex.Lock()
	defer connectionMutex.Unlock()

	if channel != nil {
		channel.Close()
	}
	if connection != nil {
		connection.Close()
	}
	return output.FLB_OK
}

func logInfo(msg string) {
	log.Printf("%s", msg)
}

func logError(msg string, err error) {
	log.Printf("%s: %s", msg, err)
}

func arrayContainsString(arr []string, str string) bool {
	for _, item := range arr {
		if item == str {
			return true
		}
	}
	return false
}

func main() {
}

func loadCACert(caCertFile string) (*x509.CertPool, error) {
	caCert, err := os.ReadFile(caCertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	return caCertPool, nil
}

func initConnection(config *ConnectionConfig) error {
	connectionMutex.Lock()         // Lock the mutex to ensure exclusive access
	defer connectionMutex.Unlock() // Unlock the mutex when the function returns

	// Check if the connection is already established to avoid reinitializing
	if connection != nil && !connection.IsClosed() {
		return nil // Connection is already established, no need to reinitialize
	}

	var err error
	if config.TLSEnabled {
		connection, err = amqp.DialTLS("amqps://"+config.User+":"+config.Password+"@"+config.Host+":"+config.Port+"/"+config.VHost, config.TLSConfig)
	} else {
		connection, err = amqp.Dial("amqp://" + config.User + ":" + config.Password + "@" + config.Host + ":" + config.Port + "/" + config.VHost)
	}
	if err != nil {
		logError("Failed to establish a connection to RabbitMQ: ", err)
		return err
	}

	channel, err = connection.Channel()
	if err != nil {
		logError("Failed to open a channel: ", err)
		connection.Close()
		return err
	}

	logInfo("Established successfully a connection to the RabbitMQ-Server")

	return nil
}
