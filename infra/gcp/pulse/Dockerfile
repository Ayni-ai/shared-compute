# Tester pulse as a Cloud Run job (replaces the launchd agent that ran on a laptop).
FROM python:3.12-slim
RUN useradd --system --uid 10001 pulse
WORKDIR /app
COPY clients/python/ayni /app/clients/python/ayni
COPY scripts/tester-pulse.py /app/scripts/tester-pulse.py
USER pulse
ENTRYPOINT ["python3", "/app/scripts/tester-pulse.py"]
