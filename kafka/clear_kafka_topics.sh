#!/bin/bash

# Configuración
KAFKA_CONTAINER="kafka"
ZOOKEEPER_HOST="zookeeper:2181" # Zookeeper se conecta desde el contenedor de Kafka
BROKER="localhost:9093"         # Kafka Broker expuesto

echo "Obteniendo la lista de topics..."
# Lista todos los topics en Kafka
TOPICS=$(docker exec -it $KAFKA_CONTAINER kafka-topics.sh --bootstrap-server $BROKER --list)

if [ -z "$TOPICS" ]; then
  echo "No hay topics disponibles en Kafka."
  exit 0
fi

echo "Se encontraron los siguientes topics:"
echo "$TOPICS"
echo

# Vaciar cada topic
for TOPIC in $TOPICS; do
  echo "Limpiando mensajes del topic: $TOPIC"

  # Configurar retención a 1 ms
  docker exec -it $KAFKA_CONTAINER kafka-configs.sh --bootstrap-server $BROKER \
    --entity-type topics --entity-name "$TOPIC" \
    --alter --add-config retention.ms=1

  # Esperar unos segundos para que Kafka elimine los mensajes
  sleep 5

  # Restaurar la configuración de retención por defecto
  docker exec -it $KAFKA_CONTAINER kafka-configs.sh --bootstrap-server $BROKER \
    --entity-type topics --entity-name "$TOPIC" \
    --alter --delete-config retention.ms

  echo "Mensajes eliminados para el topic: $TOPIC"
  echo
done

echo "Proceso completado. Todos los mensajes en los topics han sido eliminados."
