#!/bin/bash

# Configuración
KAFKA_CONTAINER="kafka" # Nombre del contenedor Kafka
BROKER="localhost:9093" # Dirección del Kafka Broker expuesto

# Obtener la lista de topics
echo "Obteniendo la lista de topics..."
# Obtener la lista de topics y limpiar caracteres no deseados
TOPICS=$(docker exec -it $KAFKA_CONTAINER kafka-topics.sh --bootstrap-server $BROKER --list 2>/dev/null | tr -d '\r')

if [ -z "$TOPICS" ]; then
  echo "No hay topics disponibles en Kafka o no se pudo conectar al broker."
  exit 1
fi

# Mostrar los topics con índices
echo "Topics disponibles:"
TOPIC_LIST=()
i=1
while IFS= read -r TOPIC; do
  echo "[$i] [$TOPIC]"
  TOPIC_LIST+=("$TOPIC")
  ((i++))
done <<< "$TOPICS"

# Solicitar al usuario que seleccione un topic
echo
read -p "Selecciona el número del topic que deseas ver: " SELECTION

# Validar la selección del usuario
if ! [[ "$SELECTION" =~ ^[0-9]+$ ]] || [ "$SELECTION" -lt 1 ] || [ "$SELECTION" -gt "${#TOPIC_LIST[@]}" ]; then
  echo "Selección inválida. Por favor, elige un número entre 1 y ${#TOPIC_LIST[@]}."
  exit 1
fi

# Limpiar y seleccionar el topic
SELECTED_TOPIC=$(echo "${TOPIC_LIST[$((SELECTION - 1))]}" | tr -d '\r\n' | xargs)
echo "Nombre del topic seleccionado: '$SELECTED_TOPIC'"

# Validar si el topic existe en Kafka
if ! echo "$TOPICS" | grep -q "^$SELECTED_TOPIC\$"; then
  echo "El topic seleccionado no existe en Kafka."
  exit 1
fi

# Mostrar el número total de mensajes en el topic
echo "Calculando el número de mensajes en el topic seleccionado..."
MESSAGE_COUNT=$(docker exec -it $KAFKA_CONTAINER kafka-run-class.sh kafka.tools.GetOffsetShell \
  --broker-list $BROKER \
  --topic "$SELECTED_TOPIC" \
  --time -1 | awk -F ":" '{sum += $3} END {print sum}')

if [ -z "$MESSAGE_COUNT" ]; then
  echo "No se pudo calcular el número de mensajes del topic '$SELECTED_TOPIC'."
  exit 1
fi

echo "El número de mensajes en el topic seleccionado es: $MESSAGE_COUNT"

# Mostrar los mensajes del topic seleccionado y salir al terminar
echo "Mostrando mensajes del topic: $SELECTED_TOPIC"
docker exec -it $KAFKA_CONTAINER kafka-console-consumer.sh \
  --bootstrap-server $BROKER \
  --topic "$SELECTED_TOPIC" \
  --from-beginning \
  --max-messages "$MESSAGE_COUNT" || {
    echo "Error: No se pudo consumir mensajes del topic '$SELECTED_TOPIC'. Verifica la configuración del broker o el estado del topic."
    exit 1
  }