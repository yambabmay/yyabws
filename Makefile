include .env

.PHONY: image

image: clean
	@docker build -t ${ASSIGNMENT_IMAGE_NAME} .

clean:
	@./scripts/clean.sh

run:
	@docker run --rm -d --name  ${ASSIGNMENT_CONTAINER_NAME}  -e "ATLAS_SECRET=${ASSIGNMENT_SECRET}" -p "${HOST_PORT}":80 ${ASSIGNMENT_IMAGE_NAME}

