FROM polly-prod:latest

USER root

# Ensure team lead users
RUN useradd -s /bin/bash --create-home dog  && su -c "mkdir /home/dog/.ssh" dog
RUN useradd -s /bin/bash --create-home cat  && su -c "mkdir /home/cat/.ssh" cat
RUN useradd -s /bin/bash --create-home duck && su -c "mkdir /home/duck/.ssh" duck
RUN useradd -s /bin/bash --create-home swan && su -c "mkdir /home/swan/.ssh" swan

# create keys
RUN su -c "ssh-keygen -t rsa -N '' -f /home/dog/.ssh/id_rsa" dog
RUN su -c "ssh-keygen -t rsa -N '' -f /home/cat/.ssh/id_rsa" cat
RUN su -c "ssh-keygen -t rsa -N '' -f /home/duck/.ssh/id_rsa" duck
RUN su -c "ssh-keygen -t rsa -N '' -f /home/swan/.ssh/id_rsa" swan

# Switch to the tools dir and create users in gerrit
# (mammals are team leads, birds are not)
WORKDIR /home/gerrit/tools
RUN  cd /home/gerrit/site && \
  bin/gerrit.sh start && \
  cd /home/gerrit/tools && \
  ./create-user --username=dog --team-lead --ssh-pubkey=/home/dog/.ssh/id_rsa.pub && \
  ./create-user --username=cat --team-lead --ssh-pubkey=/home/cat/.ssh/id_rsa.pub && \
  ./create-user --username=duck --ssh-pubkey=/home/duck/.ssh/id_rsa.pub && \
  ./create-user --username=swan --ssh-pubkey=/home/swan/.ssh/id_rsa.pub && \
  cd /home/gerrit/site && \
  bin/gerrit.sh stop

# Prepare for the user
USER gerrit
WORKDIR /home/gerrit
