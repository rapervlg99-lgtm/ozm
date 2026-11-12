package api

import (
	"encoding/json"
	"io"
	"net/http"

	"ozm/backend/internal/models"
)

// checkRow — вердикт по одной конструкции: то же, что калькулятор пишет в
// колонке «Объём / масса» и в раскрывающемся трейсе.
type checkRow struct {
	ID        string  `json:"id"`
	Exclusion string  `json:"exclusion"`
	Dpr       float64 `json:"dpr"`
	Delta     float64 `json:"delta"`
	FactR     float64 `json:"factR"`
}

// handleCalcCheck — проверка сочетаний «профиль + покрытие + предел ОС + тип
// конструкции» теми же правилами, что и калькулятор: считает тот же движок, но
// результат не сохраняется и материалы не нужны. Форма распознавания вызывает
// этот метод, чтобы подсказать, что покрытие не подходит к пределу ОС.
func (s *Server) handleCalcCheck(w http.ResponseWriter, r *http.Request) {
	var req models.CalcRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(req.Groups) == 0 {
		writeErr(w, http.StatusBadRequest, "нет конструкций для проверки")
		return
	}
	// Проверка не является расчётом проекта: обязательные поля расчёта (объект,
	// адрес, согласие) здесь не требуются, иначе форма распознавания не смогла бы
	// спросить движок до заполнения этих полей.
	req.Consent = true
	if req.ObjectName == "" {
		req.ObjectName = "проверка покрытия"
	}
	if req.Address == "" {
		req.Address = "—"
	}

	resp, err := s.engine.Calculate(req)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	out := make([]checkRow, 0, len(resp.Elements))
	for _, el := range resp.Elements {
		out = append(out, checkRow{
			ID: el.ID, Exclusion: el.Exclusion, Dpr: el.Dpr, Delta: el.Delta, FactR: el.FactR,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"elements": out})
}
