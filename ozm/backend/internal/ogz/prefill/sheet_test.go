package prefill

import (
	"math"
	"testing"
)

func TestSheetSizeFromMark(t *testing.T) {
	// «4×1250×2500» — толщина и ширина листа
	th, w := sheetSize("4×1250×2500", 31.4)
	if math.Abs(th-4) > 1e-9 || math.Abs(w-1250) > 1e-9 {
		t.Fatalf("толщина/ширина = %v/%v, ожидали 4/1250", th, w)
	}
	// «Лист 4 мм» — только толщина, ширина по умолчанию
	th, w = sheetSize("Лист 4 мм", 31.4)
	if math.Abs(th-4) > 1e-9 || math.Abs(w-1000) > 1e-9 {
		t.Fatalf("толщина/ширина = %v/%v, ожидали 4/1000", th, w)
	}
	// толщины в обозначении нет — берём из массы 1 м² (7,85 кг/м² на 1 мм)
	th, w = sheetSize("лист", 47.1)
	if math.Abs(th-6) > 0.11 || math.Abs(w-1000) > 1e-9 {
		t.Fatalf("толщина/ширина = %v/%v, ожидали ≈6/1000", th, w)
	}
}

func TestSheetGoesToCalculatorAsPlate(t *testing.T) {
	payload := Build("job-1", []RowInput{
		{
			ProfileMark: "4×1250×2500", IsSheet: true, AreaM2: 25,
			Status: "Посчитано", HeatingSides: "сверху, снизу",
			FireLimit: "R60", BearingType: "Несущая", CoatingType: "ТЕХНО ОЗМ",
			MassPerM2: 31.4,
		},
	})
	if len(payload.Items) != 1 {
		t.Fatalf("ожидали один элемент листа, получили %d", len(payload.Items))
	}
	it := payload.Items[0]
	if it.Shape != "plate" {
		t.Fatalf("вид сечения = %q, ожидали plate", it.Shape)
	}
	if it.Dims["t"] != 4 || it.Dims["b"] != 1250 {
		t.Fatalf("размеры = %v, ожидали t=4 b=1250", it.Dims)
	}
	// длина подобрана так, чтобы площадь совпала с площадью листа
	if math.Abs(it.LengthM-20) > 1e-9 {
		t.Fatalf("длина = %v, ожидали 20 м (25 м² / 1,25 м)", it.LengthM)
	}
	if payload.Sheet.AreaM2 != 25 {
		t.Fatalf("справочная площадь = %v, ожидали 25", payload.Sheet.AreaM2)
	}
	// лист без предела ОС остаётся только справочной площадью
	payload = Build("job-2", []RowInput{{ProfileMark: "4×1250", IsSheet: true, AreaM2: 10, Status: "Посчитано"}})
	if len(payload.Items) != 0 || payload.Sheet.AreaM2 != 10 {
		t.Fatalf("без R лист не должен уходить элементом: items=%d sheet=%v", len(payload.Items), payload.Sheet.AreaM2)
	}
}
