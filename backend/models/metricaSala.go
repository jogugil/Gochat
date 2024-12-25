package models

import (
	"backend/entities"
	"time"
)

type MetricaSala struct {
	MensajesEnviados    int64     `json:"mensajesEnviados"`
	UsuariosPorHora     int64     `json:"usuariosPorHora"`
	MensajesPorHora     int64     `json:"mensajesPorHora"`
	UltimaActualizacion time.Time `json:"ultimaActualizacion"`
}

func (metrica *MetricaSala) ActualizarMetricasUsuario(usuario entities.User) {
	// Incrementar el número de usuarios activos
	metrica.UsuariosPorHora++
}

func (metrica *MetricaSala) ActualizarMetricasMensaje(mensaje entities.Message) {
	// Incrementar el número de mensajes enviados
	metrica.MensajesPorHora++
	metrica.MensajesEnviados++
}

func (metrica *MetricaSala) ObtenerResumen() map[string]int64 {
	resumen := make(map[string]int64)
	resumen["mensajesEnviados"] = metrica.MensajesEnviados
	resumen["usuariosPorHora"] = metrica.UsuariosPorHora
	resumen["mensajesPorHora"] = metrica.MensajesPorHora
	return resumen
}
