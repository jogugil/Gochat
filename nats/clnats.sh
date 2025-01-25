#!/bin/bash

# Variables
NATS_SERVER="nats://localhost:4222"  # Cambia según sea necesario
NATS_CLI="nats"                     # Comando NATS CLI

# Verificar si NATS CLI está instalado
if ! command -v $NATS_CLI &>/dev/null; then
    echo "NATS CLI no está instalado. Instálalo desde: https://docs.nats.io/"
    exit 1
fi

# Listar tópicos disponibles
echo "Conectando al servidor NATS en $NATS_SERVER..."
echo "Obteniendo la lista de tópicos disponibles..."
topics=$($NATS_CLI --server $NATS_SERVER stream ls 2>/dev/null | awk '{print $1}' | tail -n +3)

if [ -z "$topics" ]; then
    echo "No se encontraron tópicos disponibles en el servidor."
    exit 1
fi

echo "Tópicos disponibles:"
select topic in $topics; do
    if [ -n "$topic" ]; then
        echo "Has seleccionado el tópico: $topic"
        break
    else
        echo "Por favor, selecciona un número válido."
    fi
done

# Subscribirse y mostrar mensajes
echo "Suscribiéndose al tópico '$topic'..."
echo "Mostrando mensajes, presiona Ctrl+C para salir."
$NATS_CLI --server $NATS_SERVER sub "$topic"
