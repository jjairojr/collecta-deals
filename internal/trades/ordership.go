package trades

func AllocateOrderShipping(total float64, lots []Trade) map[string]float64 {
	out := make(map[string]float64, len(lots))
	if len(lots) == 0 {
		return out
	}
	weights := make([]float64, len(lots))
	sum := 0.0
	for i, t := range lots {
		weights[i] = float64(qtyOf(t)) * t.BuyBRL
		sum += weights[i]
	}
	if sum <= 0 {
		sum = 0
		for i, t := range lots {
			weights[i] = float64(qtyOf(t))
			sum += weights[i]
		}
	}
	if total <= 0 || sum <= 0 {
		for _, t := range lots {
			out[t.ID] = 0
		}
		return out
	}
	given := 0.0
	largest := 0
	for i, t := range lots {
		share := round2(total * weights[i] / sum)
		out[t.ID] = share
		given += share
		if weights[i] > weights[largest] {
			largest = i
		}
	}
	out[lots[largest].ID] = round2(out[lots[largest].ID] + total - given)
	return out
}

func qtyOf(t Trade) int {
	if t.Qty <= 0 {
		return 1
	}
	return t.Qty
}
