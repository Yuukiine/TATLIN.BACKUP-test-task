package core

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"dns-manager/models"
)

type Server struct {
	log *slog.Logger
	m   *Manager
}

func NewServer(log *slog.Logger, m *Manager) *Server {
	return &Server{log: log, m: m}
}

func (s *Server) HandleAdd() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m models.DNS
		err := json.NewDecoder(r.Body).Decode(&m)
		if err != nil {
			s.log.Error("Failed to parse request body", "error", err)
			http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
			return
		}

		if net.ParseIP(m.IP) == nil {
			s.log.Error("Failed to parse IP")
			http.Error(w, "Неверный IP адрес", http.StatusBadRequest)
			return
		}
		err = s.m.Add(m.IP, m.Domain)

		if err != nil {
			if errors.Is(err, ErrServerExists) {
				s.log.Error("DNS server is already on configuration", "error", err)
				http.Error(w, "Сервер уже существует", http.StatusConflict)
				return
			}
			s.log.Error("Failed to add DNS", "error", err)
			http.Error(w, "Не удалось добавить сервер", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

func (s *Server) HandleRemove() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m models.DNS
		err := json.NewDecoder(r.Body).Decode(&m)
		if err != nil {
			s.log.Error("Failed to parse request body", "error", err)
			http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
			return
		}

		if net.ParseIP(m.IP) == nil {
			s.log.Error("Failed to parse IP")
			http.Error(w, "Неверный IP адрес", http.StatusBadRequest)
			return
		}
		err = s.m.Remove(m.IP)

		if err != nil {
			if errors.Is(err, ErrServerNotFound) {
				s.log.Error("DNS server is not found", "domain", m.Domain)
				http.Error(w, "Этот сервер не существует", http.StatusNotFound)
				return
			}
			s.log.Error("Failed to remove DNS", "error", err)
			http.Error(w, "Не удалось удалить выбранный сервер", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}

func (s *Server) HandleList() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := s.m.List(w); err != nil {
			s.log.Error("Failed to list DNS", "error", err)
			http.Error(w, "Не удалось показать список серверов", http.StatusInternalServerError)
			return
		}
	})
}
