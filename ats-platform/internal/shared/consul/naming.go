package consul

import "fmt"

const (
	ResumeServiceBaseName    = "resume-service"
	InterviewServiceBaseName = "interview-service"
	SearchServiceBaseName    = "search-service"
)

type Protocol string

const (
	ProtocolHTTP Protocol = "http"
	ProtocolGRPC Protocol = "grpc"
)

type Endpoint struct {
	BaseName string
	Protocol Protocol
	IP       string
	Port     int
}

func ServiceName(baseName string, protocol Protocol) string {
	return fmt.Sprintf("%s-%s", baseName, protocol)
}
