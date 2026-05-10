package core

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dns-manager/models"
)

func newTestServer(t *testing.T, initialContent string) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "resolv.conf")

	if err := os.WriteFile(path, []byte(initialContent), 0644); err != nil {
		t.Fatalf("не удалось создать временный конфиг: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := NewManager(path)
	return NewServer(logger, m), path
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("не удалось сериализовать тело запроса: %v", err)
	}
	return b
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("не удалось прочитать файл %s: %v", path, err)
	}
	return string(b)
}

func TestNewServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := NewManager("some/path")

	s := NewServer(logger, m)

	if s == nil {
		t.Fatal("NewServer вернул nil")
	}
	if s.log != logger {
		t.Error("NewServer не сохранил logger")
	}
	if s.m != m {
		t.Error("NewServer не сохранил manager")
	}
}

func TestHandleAdd_Success(t *testing.T) {
	s, path := newTestServer(t, "")

	body := mustMarshal(t, models.DNS{IP: "8.8.8.8", Domain: "google-dns"})
	req := httptest.NewRequest(http.MethodPost, "/add", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	s.HandleAdd().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ожидался статус %d, получен %d (%s)", http.StatusOK, rr.Code, rr.Body.String())
	}

	got := readFile(t, path)
	want := "8.8.8.8 google-dns\n"
	if got != want {
		t.Errorf("содержимое файла не совпадает.\nожидалось:\n%q\nполучено:\n%q", want, got)
	}
}

func TestHandleAdd_AppendsToExistingFile(t *testing.T) {
	s, path := newTestServer(t, "1.1.1.1 cloudflare\n")

	body := mustMarshal(t, models.DNS{IP: "8.8.8.8", Domain: "google-dns"})
	req := httptest.NewRequest(http.MethodPost, "/add", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	s.HandleAdd().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ожидался статус %d, получен %d", http.StatusOK, rr.Code)
	}

	got := readFile(t, path)
	if !strings.Contains(got, "1.1.1.1 cloudflare") {
		t.Errorf("исходная запись пропала из файла:\n%s", got)
	}
	if !strings.Contains(got, "8.8.8.8 google-dns") {
		t.Errorf("новая запись не была добавлена в файл:\n%s", got)
	}
}

func TestHandleAdd_InvalidJSON(t *testing.T) {
	s, _ := newTestServer(t, "")

	req := httptest.NewRequest(http.MethodPost, "/add", strings.NewReader("это не JSON"))
	rr := httptest.NewRecorder()

	s.HandleAdd().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("ожидался статус %d для невалидного JSON, получен %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestHandleAdd_EmptyBody(t *testing.T) {
	s, _ := newTestServer(t, "")

	req := httptest.NewRequest(http.MethodPost, "/add", strings.NewReader(""))
	rr := httptest.NewRecorder()

	s.HandleAdd().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("ожидался статус %d для пустого тела, получен %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestHandleAdd_InvalidIP(t *testing.T) {
	cases := []struct {
		name string
		ip   string
	}{
		{"пустая строка", ""},
		{"явно некорректный IP", "не-ip-адрес"},
		{"число вне диапазона", "999.999.999.999"},
		{"только три октета", "1.2.3"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newTestServer(t, "")
			body := mustMarshal(t, models.DNS{IP: tc.ip, Domain: "example"})
			req := httptest.NewRequest(http.MethodPost, "/add", bytes.NewReader(body))
			rr := httptest.NewRecorder()

			s.HandleAdd().ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("ожидался статус %d для IP %q, получен %d", http.StatusBadRequest, tc.ip, rr.Code)
			}
		})
	}
}

func TestHandleRemove_Success(t *testing.T) {
	s, path := newTestServer(t, "8.8.8.8 google-dns\n1.1.1.1 cloudflare\n")

	body := mustMarshal(t, models.DNS{IP: "8.8.8.8", Domain: "google-dns"})
	req := httptest.NewRequest(http.MethodDelete, "/add", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	s.HandleRemove().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("ожидался статус %d, получен %d (%s)", http.StatusNoContent, rr.Code, rr.Body.String())
	}

	got := readFile(t, path)
	if strings.Contains(got, "8.8.8.8") {
		t.Errorf("запись 8.8.8.8 всё ещё в файле:\n%s", got)
	}
	if !strings.Contains(got, "1.1.1.1 cloudflare") {
		t.Errorf("остальные записи были удалены:\n%s", got)
	}
}

func TestHandleRemove_InvalidJSON(t *testing.T) {
	s, _ := newTestServer(t, "8.8.8.8 google-dns\n")

	req := httptest.NewRequest(http.MethodDelete, "/add", strings.NewReader("{кривой json"))
	rr := httptest.NewRecorder()

	s.HandleRemove().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("ожидался статус %d, получен %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestHandleList_Success(t *testing.T) {
	content := "8.8.8.8 google-dns\n1.1.1.1 cloudflare\n"
	s, _ := newTestServer(t, content)

	req := httptest.NewRequest(http.MethodGet, "/list", nil)
	rr := httptest.NewRecorder()

	s.HandleList().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ожидался статус %d, получен %d", http.StatusOK, rr.Code)
	}
	if got := rr.Body.String(); got != content {
		t.Errorf("тело ответа не совпадает.\nожидалось:\n%q\nполучено:\n%q", content, got)
	}
}

func TestHandleList_EmptyFile(t *testing.T) {
	s, _ := newTestServer(t, "")

	req := httptest.NewRequest(http.MethodGet, "/list", nil)
	rr := httptest.NewRecorder()

	s.HandleList().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ожидался статус %d для пустого файла, получен %d", http.StatusOK, rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Errorf("ожидалось пустое тело, получено: %q", rr.Body.String())
	}
}
