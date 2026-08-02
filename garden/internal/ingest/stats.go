package ingest

import "context"

type Stats struct {
	Accepted  int `json:"accepted"`
	Running   int `json:"running"`
	Spooled   int `json:"spooled"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Total     int `json:"total"`
}

func (s *Service) Stats(ctx context.Context) (Stats, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM ingestions GROUP BY status`)
	if err != nil {
		return Stats{}, err
	}
	defer rows.Close()

	var st Stats
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return Stats{}, err
		}
		switch status {
		case "accepted":
			st.Accepted = count
		case "running":
			st.Running = count
		case "spooled":
			st.Spooled = count
		case "completed":
			st.Completed = count
		case "failed":
			st.Failed = count
		}
		st.Total += count
	}
	return st, rows.Err()
}
