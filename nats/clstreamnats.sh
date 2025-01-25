#!/bin/bash

# Configuración del servidor NATS
NATS_SERVER="ws://localhost:8080"  # Cambia si usas otro puerto o servidor
NATS_USER_ADMIN="admin"  # Cambia esto por tu usuario de NATS
NATS_PASS_ADMIN="admin"  # Cambia esto por tu contraseña de NATS
NATS_USER="user"  # Cambia esto por tu usuario de NATS
NATS_PASS="user123"  # Cambia esto por tu contraseña de NATS
NATS_CLI="nats"  # Comando del cliente NATS

# Verificar si NATS CLI está instalado
if ! command -v $NATS_CLI &>/dev/null; then
    echo "NATS CLI no está instalado. Instálalo desde: https://docs.nats.io/"
    exit 1
fi

# Modificación del menú principal
function main_menu() {
    echo "-----------------------------"
    echo " Menú principal - NATS "
    echo "-----------------------------"
    echo "1) Ver estado del servidor"
    echo "2) Listar tópicos"
    echo "3) Eliminar todos los tópicos"
    echo "4) Salir"
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
        delete_all_topics
        ;;
    4)
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
    echo "-----------------------------"
    echo " Selecciona el tipo de información que deseas ver:"
    echo "1) Conexiones"
    echo "2) Jetstream"
    echo "3) Información general"
    echo "-----------------------------"
    read -p "Selecciona una opción: " option

    case $option in
    1)
        $NATS_CLI --server $NATS_SERVER --user $NATS_USER_ADMIN --password $NATS_PASS_ADMIN server report connections
        ;;
    2)
        $NATS_CLI --server $NATS_SERVER --user $NATS_USER_ADMIN --password $NATS_PASS_ADMIN server report jetstream
        ;;
    3)
        $NATS_CLI --server $NATS_SERVER --user $NATS_USER_ADMIN --password $NATS_PASS_ADMIN server info
        ;;
    *)
        echo "Opción no válida, intenta de nuevo."
        server_status
        ;;
    esac
    main_menu
}

function list_topics() {
    echo "Listando los streams (tópicos de JetStream) disponibles..."
    
    # Traza: Verificar los parámetros de conexión y el comando que se está ejecutando
    echo "Comando ejecutado: \$NATS_CLI --server $NATS_SERVER --user $NATS_USER --password $NATS_PASS stream ls --json"
    
    # Ejecutar el comando para listar los streams
    streams=$($NATS_CLI --server $NATS_SERVER --user $NATS_USER --password $NATS_PASS stream ls --json 2>/dev/null)
    
    # Traza: Mostrar la salida bruta del comando
    echo "Salida del comando 'stream ls':"
    echo "$streams"
    
    # Filtrar los nombres de los streams (modificación para el array de strings)
    streams_names=$(echo "$streams" | jq -r '.[]')
    
    # Traza: Mostrar los nombres de los streams obtenidos
    echo "Streams obtenidos:"
    echo "$streams_names"
    
    # Verificar si la variable streams está vacía
    if [ -z "$streams_names" ]; then
        echo "No se encontraron streams disponibles."
        main_menu
        return
    fi

    echo "Streams disponibles:"
    
    # Traza: Ver qué streams serán mostrados
    echo "Streams a mostrar:"
    echo "$streams_names"
    
    # Mostrar el menú de selección de streams
    select stream in $streams_names; do
        if [ -n "$stream" ]; then
            echo "Has seleccionado el stream: $stream"
            topic_menu "$stream"
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
    echo "3) Enviar un mensaje al tópico"
    echo "4) Eliminar tópico"
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
        send_message "$topic"
        ;;
    4)
        delete_topic "$topic"
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
    
    # Habilitar trazas para ver los comandos que se están ejecutando
    set -x
    
   
   
    
    # Mostrar los mensajes con el consumidor creado
    echo "Mostrando los mensajes del consumidor '$topic-consumer'..."
    $NATS_CLI --server $NATS_SERVER --user $NATS_USER --password $NATS_PASS stream view  "$topic" 
    
    # Deshabilitar trazas después de ejecutar los comandos
    set +x
    
    topic_menu "$topic"
}

# Función para limpiar todos los mensajes de un tópico
function clear_messages() {
    local topic=$1
    echo "Limpiando todos los mensajes del tópico '$topic'..."
    
    # Habilitar trazas para ver los comandos que se están ejecutando
    set -x
    
    $NATS_CLI --server $NATS_SERVER --user $NATS_USER --password $NATS_PASS stream purge "$topic"
    
    echo "Todos los mensajes del tópico '$topic' han sido eliminados."
    
    # Deshabilitar trazas después de ejecutar los comandos
    set +x
    
    topic_menu "$topic"
}

# Función para enviar un mensaje al tópico
function send_message() {
    local topic=$1
    read -p "Introduce el mensaje que deseas enviar: " message
    echo "Enviando el mensaje '$message' al tópico '$topic'..."
    
    # Habilitar trazas para ver los comandos que se están ejecutando
    set -x
    
    $NATS_CLI --server $NATS_SERVER --user $NATS_USER --password $NATS_PASS pub "$topic" "$message"
    
    echo "Mensaje enviado al tópico '$topic'."
    
    # Deshabilitar trazas después de ejecutar los comandos
    set +x
    
    topic_menu "$topic"
}

# Función para eliminar un tópico
function delete_topic() {
    local topic=$1
    read -p "¿Estás seguro de que deseas eliminar el tópico '$topic'? (s/n): " confirmation
    if [[ "$confirmation" == "s" || "$confirmation" == "S" ]]; then
        echo "Eliminando el tópico '$topic'..."
        
        # Habilitar trazas para ver los comandos que se están ejecutando
        set -x
        
        $NATS_CLI --server $NATS_SERVER --user $NATS_USER --password $NATS_PASS stream delete "$topic"
        
        echo "Tópico '$topic' eliminado."
    else
        echo "Eliminación cancelada."
    fi
    
    # Deshabilitar trazas después de ejecutar los comandos
    set +x
    
    main_menu
}
# Función para eliminar todos los tópicos (streams)
function delete_all_topics() {
    echo "Obteniendo la lista de todos los streams disponibles..."

    # Habilitar trazas para verificar los comandos ejecutados
    set -x

    # Listar todos los streams
    streams=$($NATS_CLI --server $NATS_SERVER --user $NATS_USER --password $NATS_PASS stream ls --json 2>/dev/null)

    # Deshabilitar trazas temporalmente
    set +x

    # Mostrar la salida bruta del comando para depuración
    echo "Salida del comando 'stream ls':"
    echo "$streams"

    # Obtener los nombres de los streams
    streams_names=$(echo "$streams" | jq -r '.[]')

    # Mostrar los nombres de los streams obtenidos
    echo "Streams disponibles:"
    echo "$streams_names"

    # Verificar si hay streams disponibles
    if [ -z "$streams_names" ]; then
        echo "No se encontraron streams disponibles."
        main_menu
        return
    fi

    # Confirmar antes de eliminar todos los streams
    read -p "¿Estás seguro de que deseas eliminar TODOS los streams? (s/n): " confirmation
    if [[ "$confirmation" != "s" && "$confirmation" != "S" ]]; then
        echo "Eliminación de todos los streams cancelada."
        main_menu
        return
    fi

    # Eliminar cada stream encontrado
    for stream in $streams_names; do
        echo "Eliminando stream '$stream'..."
        
        # Habilitar trazas para mostrar comandos ejecutados
        set -x
        
        $NATS_CLI --server $NATS_SERVER --user $NATS_USER --password $NATS_PASS stream delete "$stream"

        # Deshabilitar trazas después de cada eliminación
        set +x
    done

    echo "Todos los streams han sido eliminados."
    main_menu
}
# Iniciar el script
main_menu