FROM python:3.8-slim

RUN pip install ast2json

WORKDIR /opt

COPY api.py run.py ./

CMD python3 run.py
