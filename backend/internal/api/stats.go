package api

import (
	"net/http"
)

type statsResponse struct {
	ByStatus   map[string]int `json:"by_status"`
	OpenByPrio map[string]int `json:"open_by_priority"`
	Unassigned int            `json:"unassigned_open"`
	// Horas promedio entre la creación y la resolución, en los últimos 30 días.
	AvgResolutionHours *float64 `json:"avg_resolution_hours_30d"`
	// Tickets pendientes que ya superaron algún plazo de SLA.
	SLABreached int `json:"sla_breached"`
	// Valoraciones de los clientes en los últimos 30 días.
	Satisfaction struct {
		Good int `json:"good"`
		Bad  int `json:"bad"`
	} `json:"satisfaction_30d"`
}

// stats calcula los números del dashboard en una sola consulta.
func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	resp := statsResponse{
		ByStatus:   map[string]int{},
		OpenByPrio: map[string]int{},
	}
	for _, st := range ticketStatuses {
		resp.ByStatus[st] = 0
	}
	for _, p := range ticketPriorities {
		resp.OpenByPrio[p] = 0
	}

	rows, err := s.db.Query(r.Context(), `
		SELECT 'status' AS kind, status AS key, count(*) FROM tickets WHERE org_id = $1 GROUP BY status
		UNION ALL
		SELECT 'priority', priority, count(*) FROM tickets
		WHERE org_id = $1 AND status NOT IN ('resolved', 'closed') GROUP BY priority
		UNION ALL
		SELECT 'unassigned', '', count(*) FROM tickets
		WHERE org_id = $1 AND assignee_id IS NULL AND status NOT IN ('resolved', 'closed')`, claimsFrom(r).OrgID)
	if err != nil {
		internalError(w, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var kind, key string
		var n int
		if err := rows.Scan(&kind, &key, &n); err != nil {
			internalError(w, err)
			return
		}
		switch kind {
		case "status":
			resp.ByStatus[key] = n
		case "priority":
			resp.OpenByPrio[key] = n
		case "unassigned":
			resp.Unassigned = n
		}
	}
	if err := rows.Err(); err != nil {
		internalError(w, err)
		return
	}

	if err := s.db.QueryRow(r.Context(), `
		SELECT
			(SELECT avg(extract(epoch FROM resolved_at - created_at) / 3600)
			 FROM tickets WHERE org_id = $1 AND resolved_at > now() - interval '30 days'),
			(SELECT count(*) FROM tickets t JOIN sla_policies p ON p.org_id = t.org_id AND p.priority = t.priority
			 WHERE t.org_id = $1 AND `+slaBreached+`),
			(SELECT count(*) FROM tickets WHERE org_id = $1 AND satisfaction = 'good' AND rated_at > now() - interval '30 days'),
			(SELECT count(*) FROM tickets WHERE org_id = $1 AND satisfaction = 'bad' AND rated_at > now() - interval '30 days')`,
		claimsFrom(r).OrgID,
	).Scan(&resp.AvgResolutionHours, &resp.SLABreached, &resp.Satisfaction.Good, &resp.Satisfaction.Bad); err != nil {
		internalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
