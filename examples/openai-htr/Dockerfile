ARG TAG=main
ARG DOCKER_REPOSITORY=lehighlts
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

ENV OPENAI_MODEL=gpt-4o-mini
ENV PROMPT="Transcribe this image that contains handwritten text. Include all text you see in the image. In your response, say absolutely nothing except the text from the image"
COPY scyllaridae.yml /app/scyllaridae.yml
