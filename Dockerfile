FROM python:3

ADD ./src/*.py /
ADD ./requirements.txt /requirements.txt

RUN pip install -r /requirements.txt

CMD [ "python", "./index.py" ]
