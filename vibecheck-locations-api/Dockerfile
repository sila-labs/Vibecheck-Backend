FROM python:3.7
COPY requirements.txt /opt/app/requirements.txt

WORKDIR /opt/app

RUN pip install -r requirements.txt
COPY . /opt/app

EXPOSE 6464


CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "6464"]
