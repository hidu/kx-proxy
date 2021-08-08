// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/8/7

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var DefaultReg = prometheus.NewRegistry()
