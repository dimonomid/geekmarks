package postgres

import (
	"fmt"

	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

type childrenCheck struct {
	id                int
	childrenCnt       int
	childrenCntActual int
}

func (cc *childrenCheck) String() string {
	return fmt.Sprintf("id=%d, childrenCnt=%d, childrenCntActual=%d",
		cc.id, cc.childrenCnt, cc.childrenCntActual,
	)
}

func (s *StoragePostgres) CheckIntegrity() error {
	err := s.checkChildrenCnt()
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (s *StoragePostgres) checkChildrenCnt() error {
	rows, err := s.db.Query(`
SELECT id, children_cnt, children_cnt_actual
  FROM
    (SELECT id, children_cnt,
            (SELECT COUNT(id) FROM tags WHERE parent_id = t.id) AS children_cnt_actual
          FROM tags t) T
  WHERE children_cnt != children_cnt_actual
`,
	)
	if err != nil {
		return errors.Trace(err)
	}

	failures := []childrenCheck{}
	str := ""

	defer rows.Close()
	for rows.Next() {
		var cur childrenCheck
		err := rows.Scan(&cur.id, &cur.childrenCnt, &cur.childrenCntActual)
		if err != nil {
			return errors.Trace(err)
		}

		failures = append(failures, cur)
		str += cur.String() + "\n"
	}

	if len(str) > 0 {
		return errors.Errorf("children count integrity is broken: %s", str)
	}

	return nil
}
