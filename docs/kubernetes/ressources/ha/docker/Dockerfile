FROM updated-bionic
#FROM ubuntu:latest
COPY scripts /home/scripts
WORKDIR /home/scripts
RUN ["chmod", "+x", "startup_script.sh"]
RUN ["chmod", "+x", "haproxy.sh"]
RUN ["chmod", "+x", "keepalived.sh"]
RUN ["chmod", "+x", "service_launch.sh"]

ENV KUBE_MASTER_VIP=192.168.168.100
ENV KUBE_MASTER_IP_01=192.168.168.101
ENV KUBE_MASTER_IP_02=192.168.168.102
ENV KUBE_MASTER_IP_03=192.168.168.103
ENV NODE_NETWORK_TYPE=eth0

RUN ["bash","./startup_script.sh"]


# ENTRYPOINT ["sh","./start_script.sh"]

# CMD [ "echo", "yolo" ]