# fluent-bit-rabbitmq-output

With this Fluent Bit plugin, it is possible to send logs and metrics, which have been collected by Fluent Bit, to a RabbitMQ exchange. Furthermore, it is possible with this plugin to store data fields from each collected log or metric into the routing key.

## Build

Run the following command to build the plugin:
``` bash
build
```

## Configuration

### Configuration Parameters

| Parameter                    | Description                                                                                     | Default Value |  
|------------------------------|-------------------------------------------------------------------------------------------------|---------------|  
| RabbitHost                   | The hostname of the RabbitMQ server                                                             | ""            |  
| RabbitPort                   | The port under which the RabbitMQ is reachable                                                  | ""            |  
| RabbitUser                   | The user of the RabbitMQ host                                                                   | ""            |  
| RabbitPassword               | The password of the user which connects to the RabbitMQ server                                  | ""            |  
| RabbitVHost                  | The virtual host of the RabbitMQ server                                                         | ""            |  
| ExchangeName                 | The exchange to which Fluent Bit sends its logs                                                 | ""            |  
| ExchangeType                 | The exchange type                                                                               | ""            |  
| RoutingKey                   | The routing key pattern                                                                         | ""            |  
| RoutingKeyDelimiter          | The delimiter which separates the routing key parts                                             | "."           |  
| RemoveRkValuesFromRecord     | If enabled, Fluent Bit deletes the values of the record which have been stored in the routing key | ""          |  
| ContentEncoding              | Sets the content encoding if needed                                                             | ""            |  
| TLSCertFile                  | Path to the client certificate file                                                             | ""            |  
| TLSKeyFile                   | Path to the client key file                                                                     | ""            |  
| TLSCACertFile                | Path to the server's CA certificate file                                                        | ""            |  
| TLSInsecureSkipVerify        | If true, skips the validation of the remote certificate                                         | ""            |  
| TLSEnabled                   | If true, enables TLS and uses AMQPS protocol                                                    | ""            |

### Routing Key Pattern

You can access values from each record and store them into the routing key by specifying a record accessor.

#### Example Record Accessor
```conf
$['key1'][0]["key2"]
```

The routing key parts are delimited by the `RoutingKeyDelimiter`. If a string in one of the record accessors contains the delimiter, the plugin will not work as expected.

#### Example Routing Key

```conf
$["loglevel_3"].$["loglevel_1"][2]["sublevel"][0].$["loglevel_2"]["info_loglevel"]
```

This functionality has been implemented with the [Record Accessor - Plugin Helper](https://docs.fluentd.org/plugin-helper-overview/api-plugin-helper-record_accessor).

## Run

If you want to run the plugin with the example config in `/conf`, you need to run the following command:
```bash
fluent-bit -e ./out_rabbitmq.so -c conf/fluent-bit-docker.conf  
```
