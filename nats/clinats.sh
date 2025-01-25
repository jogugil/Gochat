#!/bin/bash

# Configuración del servidor NATS
NATS_SERVER="ws://localhost:8080"  # Cambia si usas otro puerto o servidor
NATS_CLI="nats"                   # Comando del cliente NATS CLI

# Verificar si NATS CLI está instalado
if ! command -v $NATS_CLI &>/dev/null; then
    echo "NATS CLI no está instalado. Instálalo desde: https://docs.nats.io/"
    exit 1
fi

# Función para mostrar el menú principal
function main_menu() {
    echo "-----------------------------"
    echo " Menú principal - NATS "
    echo "-----------------------------"
    echo "1) Ver estado del servidor"
    echo "2) Listar tópicos"
    echo "3) Salir"
    echo "-----------------------------"
    read -p "Selecciona una opción: " option

    case $option in
    1)
        server_status
        ;;
    2)
        list_topics
        ;;
    3)
        echo "Saliendo..."
        exit 0
        ;;
    *)
        echo "Opción no válida, intenta de nuevo."
        main_menu
        ;;
    esac
}

# Función para ver el estado del servidor
function server_status() {
    echo "Obteniendo el estado del servidor NATS..."
    $NATS_CLI --server $NATS_SERVER server report
    main_menu
}

# Función para listar los tópicos
function list_topics() {
    echo "Listando los tópicos disponibles..."
    topics=$($NATS_CLI --server $NATS_SERVER stream ls | awk '{print $1}' | tail -n +3)

    if [ -z "$topics" ]; then
        echo "No se encontraron tópicos disponibles."
        main_menu
        return
    fi

    echo "Tópicos disponibles:"
    select topic in $topics; do
        if [ -n "$topic" ]; then
            echo "Has seleccionado el tópico: $topic"
            topic_menu "$topic"
            break
        else
            echo "Por favor, selecciona un número válido."
        fi
    done
}

# Menú para gestionar un tópico
function topic_menu() {
    local topic=$1
    echo "-----------------------------"
    echo " Opciones para el tópico: $topic"
    echo "-----------------------------"
    echo "1) Ver mensajes del tópico"
    echo "2) Limpiar todos los mensajes"
    echo "3) Borrar un mensaje específico"
    echo "4) Enviar un mensaje al tópico"
    echo "5) Volver al menú principal"
    echo "-----------------------------"
    read -p "Selecciona una opción: " option

    case $option in
    1)
        view_messages "$topic"
        ;;
    2)
        clear_messages "$topic"
        ;;
    3)
        delete_message "$topic"
        ;;
    4)
        send_message "$topic"
        ;;
    5)
        main_menu
        ;;
    *)
        echo "Opción no válida, intenta de nuevo."
        topic_menu "$topic"
        ;;
    esac
}

# Función para ver los mensajes de un tópico
function view_messages() {
    local topic=$1
    echo "Mostrando mensajes del tópico '$topic'..."
    $NATS_CLI --server $NATS_SERVER sub "$topic" &
    sleep 5  # Esperar 5 segundos para capturar mensajes
    kill $!  # Detener la subscripción
    topic_menu "$topic"
}

# Función para limpiar todos los mensajes de un tópico
function clear_messages() {
    local topic=$1
    echo "Limpiando todos los mensajes del tópico '$topic'..."
    $NATS_CLI --server $NATS_SERVER stream purge "$topic"
    echo "Todos los mensajes del tópico '$topic' han sido eliminados."
    topic_menu "$topic"
}

# Función para borrar un mensaje específico de un tópico
function delete_message() {
    local topic=$1
    read -p "Introduce el ID del mensaje que deseas borrar: " message_id
    echo "Eliminando el mensaje con ID '$message_id' del tópico '$topic'..."
    $NATS_CLI --server $NATS_SERVER stream rm "$topic" --msg-id "$message_id"
    echo "Mensaje eliminado."
    topic_menu "$topic"
}

# Función para enviar un mensaje al tópico
function send_message() {
    local topic=$1
    read -p "Introduce el mensaje que deseas enviar: " message
    $NATS_CLI --server $NATS_SERVER pub "$topic" "$message"
    echo "Mensaje enviado al tópico '$topic'."
    topic_menu "$topic"
}

# Iniciar el script
main_menu
