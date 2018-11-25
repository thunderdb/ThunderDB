/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package shard

import (
	"context"
	"database/sql/driver"
	"errors"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/sqlparser"
)

// buildInsertPlan builds the route for an INSERT statement.
func buildInsertPlan(ins *sqlparser.Insert) (*Insert, error) {
	log.Debugf("buildInsertPlan got %#v", ins)

	if ins.Action == sqlparser.ReplaceStr {
		return nil, errors.New("unsupported: REPLACE INTO with sharded schema")
	}
	return &Insert{}, nil
}

type Insert struct {
}

func (*Insert) ExecContext(ctx context.Context) (result driver.Result, err error) {
	panic("implement me")
}
