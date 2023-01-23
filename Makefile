docker-build:
	docker build --tag echo-prometheus-demo .
docker-run:
	docker run -d --name echo-prometheus-demo --network prom-test-net -p 8080:8080 echo-prometheus-demo
	docker run -d --name prometheus --network prom-test-net -v $(HOME)/prometheus.yml:/etc/prometheus/prometheus.yml -p 9090:9090 prom/prometheus
docker-stop:
	docker stop echo-prometheus-demo prometheus
docker-rm:
	docker rm echo-prometheus-demo prometheus
