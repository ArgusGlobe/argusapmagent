package metadata

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPClientReadsTaskAndStats(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/task", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"Cluster":"arn:aws:ecs:ap-south-1:1:cluster/demo",
			"TaskARN":"arn:aws:ecs:ap-south-1:1:task/demo/task-1",
			"Family":"demo-service",
			"LaunchType":"FARGATE",
			"Containers":[{"DockerId":"c1","Name":"app","KnownStatus":"RUNNING"}]
		}`))
	})
	mux.HandleFunc("/task/stats", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"c1":{
				"cpu_stats":{"cpu_usage":{"total_usage":10},"system_cpu_usage":20,"online_cpus":1},
				"precpu_stats":{"cpu_usage":{"total_usage":5},"system_cpu_usage":10,"online_cpus":1},
				"memory_stats":{"usage":10,"limit":100}
			}
		}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.URL, server.Client(), slog.Default())
	task, err := client.Task(context.Background())
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	if task.Family != "demo-service" {
		t.Fatalf("unexpected family %q", task.Family)
	}
	stats, err := client.TaskStats(context.Background())
	if err != nil {
		t.Fatalf("task stats: %v", err)
	}
	if _, ok := stats["c1"]; !ok {
		t.Fatalf("expected c1 stats")
	}
}
