#/usr/bin/env bash
# This script is used by the Makefile clean target
. "$(dirname "${BASH_SOURCE}")/../.env"

# Stop the container if it is running
if (docker ps | grep -q ${ASSIGNMENT_CONTAINER_NAME})
then
    docker stop ${ASSIGNMENT_CONTAINER_NAME}
fi
# Delete the containet if it exits
if (docker ps | grep -q ${ASSIGNMENT_CONTAINER_NAME}); then
    docker container rm ${ASSIGNMENT_CONTAINER_NAME}
fi
# Delete the image if it exist
if (docker image ls -a | grep -q ${ASSIGNMENT_IMAGE_NAME}); then
    docker image rm ${ASSIGNMENT_IMAGE_NAME}
fi
