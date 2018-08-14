package providers

import (
	"github.com/banzaicloud/pipeline/pkg/providers/alibaba"
	"github.com/banzaicloud/pipeline/pkg/providers/amazon"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/banzaicloud/pipeline/pkg/providers/google"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle"
)

const (
	Alibaba = alibaba.Provider
	Amazon  = amazon.Provider
	Azure   = azure.Provider
	Google  = google.Provider
	Oracle  = oracle.Provider
)
