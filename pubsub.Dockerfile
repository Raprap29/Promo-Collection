FROM google/cloud-sdk:latest

# Install Java (required for Pub/Sub emulator)
RUN apt-get update && apt-get install -y default-jre && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Install Pub/Sub emulator
RUN gcloud components install pubsub-emulator --quiet

WORKDIR /data

# Expose Pub/Sub emulator port
EXPOSE 8085

# Start Pub/Sub emulator
CMD ["gcloud", "emulators", "pubsub", "start", "--host-port=0.0.0.0:8085"]
