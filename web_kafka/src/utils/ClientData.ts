
let isBlockedByAdblocker = false; // Variable global para almacenar el estado del bloqueo

export async function getClientInformation(): Promise<string> {
    // Obtener detalles del navegador
    const navegador = {
        userAgent: navigator.userAgent, // Cadena completa del navegador
        language: navigator.language,  // Idioma del navegador
        vendor: navigator.vendor,      // Proveedor del navegador
        urlActual: window.location.href, // URL actual del sitio
    };
    
    // Función para obtener la IP y país
    const getLocation = async (): Promise<{ ip: string; pais: string }> => {
        if (isBlockedByAdblocker) {
            console.log("La solicitud ya fue bloqueada por un bloqueador de anuncios.");
            return { ip: 'Desconocida', pais: 'Desconocido' };
        }
    
        try {
            const respuesta = await fetch('http://ip-api.com/json');
            if (!respuesta.ok) {
                throw new Error('No se pudo obtener la respuesta de la API');
            }
    
            const datos = await respuesta.json();
            const ip = datos.query || 'Desconocida';
            const pais = datos.country || 'Desconocido';
    
            return { ip, pais };
        } catch (error: any) {
            if (error instanceof Error && error.message.includes("ERR_BLOCKED_BY_ADBLOCKER")) {
                console.log("Solicitud bloqueada por un bloqueador de anuncios.");
                isBlockedByAdblocker = true; // Recordar que se bloqueó
            } else {
                console.log('Error al obtener la ubicación:', error.message || error);
            }
    
            return { ip: 'Desconocida', pais: 'Desconocido' };
        }
    };
    // Llamada para obtener IP y país
    const { ip, pais } = await getLocation();

    // Crear un objeto con toda la información del cliente
    const informacionCliente = {
        navegador: navegador,
        ip: ip,
        pais: pais,
    };

    // Convertir la información a JSON formateado
    return JSON.stringify(informacionCliente, null, 2);
}