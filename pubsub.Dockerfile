FROM google/cloud-sdk:latest

RUN apt-get update \
    && apt-get install -y --no-install-recommends default-jre-headless google-cloud-cli-pubsub-emulator \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && rm -rf /root/.config/gcloud/logs

WORKDIR /data

EXPOSE 8085

CMD ["gcloud", "beta", "emulators", "pubsub", "start", "--host-port=0.0.0.0:8085"]