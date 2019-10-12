package sqlstore

import (
	"strings"
	"testing"

	"github.com/go-xorm/xorm"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchBuilder_TagFilter(t *testing.T) {
	signedInUser := &models.SignedInUser{
		OrgId:  1,
		UserId: 1,
	}

	engine, err := xorm.NewEngine("postgres", "dbname=foo")
	require.Nil(t, err)

	dialect := migrator.NewDialect(engine)

	sb := NewSearchBuilder(signedInUser, 1000, 0, models.PERMISSION_VIEW, dialect)

	sql, params := sb.WithTags([]string{"tag1", "tag2"}).ToSql()

	assert.True(t, strings.HasPrefix(sql, "SELECT"))
	assert.True(t, strings.Contains(sql, "LEFT OUTER JOIN dashboard_tag"))
	assert.True(t, strings.Contains(sql, "ORDER BY dashboard.title ASC"))
	assert.Greater(t, len(params), 0)
}

func TestSearchBuilder_Normal(t *testing.T) {
	signedInUser := &models.SignedInUser{
		OrgId:  1,
		UserId: 1,
	}

	engine, err := xorm.NewEngine("postgres", "dbname=foo")
	require.Nil(t, err)

	dialect := migrator.NewDialect(engine)

	sb := NewSearchBuilder(signedInUser, 1000, 0, models.PERMISSION_VIEW, dialect)

	sql, params := sb.IsStarred().WithTitle("test").ToSql()

	assert.True(t, strings.HasPrefix(sql, "SELECT"))
	assert.True(t, strings.Contains(sql, "INNER JOIN dashboard on ids.id = dashboard.id"))
	assert.True(t, strings.Contains(sql, "ORDER BY dashboard.title ASC"))
	assert.Greater(t, len(params), 0)
}
