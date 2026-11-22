package handler

import (
	"os"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/testdb"
)

func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }
