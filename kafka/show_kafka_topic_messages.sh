#!/bin/bash
# Configuración
KAFKA_CONTAINER="kafka" # Nombre del contenedor Kafka
BROKER="localhost:9093" # Dirección del Kafka Broker expuesto

# Listar los topics disponibles
echo "Obteniendo la lista de topics..."
TOPICS=$(docker exec -it $KAFKA_CONTAINER kafka-topics.sh --bootstrap-server $BROKER --list 2>/dev/null | tr -d '\r')

if [ -z "$TOPICS" ]; then
  echo "No hay topics disponibles en Kafka o no se pudo conectar al broker."
  exit 1
fi

# Mostrar todos los topics sin filtrar (para depuración)
echo "* Topics obtenidos:"
echo "$TOPICS"

# Filtrar los topics que terminan en .client o .server
echo "* Filtrando topics válidos..."
TOPIC_LIST=()
while IFS= read -r TOPIC; do
  if [[ "$TOPIC" == *".client" || "$TOPIC" == *".server" ]]; then
    TOPIC_LIST+=("$TOPIC")
  fi
done <<< "$TOPICS"

if [ ${#TOPIC_LIST[@]} -eq 0 ]; then
  echo "No hay topics válidos para procesar."
  exit 0
fi

# Recorrer todos los topics y mostrar mensajes
echo "Recorriendo todos los topics y mostrando mensajes..."

for TOPIC in "${TOPIC_LIST[@]}"; do
  echo "### Mostrando mensajes del topic: $TOPIC ###"

  # Capturar mensajes del topic y filtrar errores y líneas vacías
  MESSAGES=$(docker exec -it $KAFKA_CONTAINER kafka-console-consumer.sh \
    --bootstrap-server $BROKER \
    --topic "$TOPIC" \
    --from-beginning \
    --timeout-ms 5000 \
    --max-messages 10 2>&1 | grep -E '^{.*}$')

  if [ -z "$MESSAGES" ]; then
    echo "[INFO] No hay mensajes válidos en el topic: $TOPIC"
  else
    echo "$MESSAGES"
  fi

  echo "Mensajes mostrados del topic: $TOPIC"
done

echo "Finalizado el recorrido de los topics."
