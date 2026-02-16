package gorag_qdrant

import (
	"context"
	"log"

	"github.com/qdrant/go-client/qdrant"
)

const (
	QdrantDefaulHost  string = "localhost"
	QdrantDefaultPort int    = 6334
)

type QdrantRag struct {
	QdrantUriHost string
	QdrantUriPort int
	QdrantClient  *qdrant.Client
}

func init() {
}

func NewQdrantRag(host string, port int) (q *QdrantRag, err error) {
	if len(host) == 0 {
		host = QdrantDefaulHost
	}

	if port <= 0 {
		port = QdrantDefaultPort
	}

	log.Printf("[NewQdrantRag] qdrant: host is '%s', port is %d\n", host, port)

	q = &QdrantRag{
		QdrantUriHost: host,
		QdrantUriPort: port,
		QdrantClient:  nil,
	}

	q.QdrantClient, err = qdrant.NewClient(&qdrant.Config{
		Host: q.QdrantUriHost,
		Port: q.QdrantUriPort,
	})

	if err != nil {
		log.Fatalf("[NewQdrantRag] error: %s\n", err.Error())
		return nil, err
	}

	return q, nil
}

func (q *QdrantRag) checkHealth() (err error) {
	return nil
}

func (q *QdrantRag) GetPoints(collection string, text string) (err error) {
	ctx := context.Background()

	plSelector := &qdrant.WithPayloadSelector{
		SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true},
	}

	queryVector := []float32{0, 0, 0, 0}

	// client, err := qdrant.NewClient()
	sr, err := q.QdrantClient.GetPointsClient().Search(ctx, &qdrant.SearchPoints{
		CollectionName: collection,
		Vector:         queryVector,
		Limit:          3,
		WithPayload:    plSelector,
	})

	if err != nil {
		return err
	}

	for _, point := range sr.Result {
		log.Printf("[GetPoints] found point: [%v, %f]\n", point.Id, point.Score)
	}

	return nil
}
