FROM polly-base:latest

# Ensure users
RUN useradd -s /bin/bash -u 1000 --create-home admin
RUN useradd -s /bin/bash -u 1001 --create-home gerrit --groups admin

# setup admin user ssh keys
USER admin
WORKDIR /home/admin
RUN mkdir .ssh && ssh-keygen -t rsa -N '' -f .ssh/id_rsa

# Copy in gerrit files
USER    gerrit
WORKDIR /home/gerrit
COPY    rootfs/home/gerrit/ ./

# Download gerrit
RUN wget https://www.gerritcodereview.com/download/gerrit-2.12.3.war -q

# Initialize gerrit
RUN java -jar gerrit-2.12.3.war init \
  --site-path site \
  --no-auto-start \
  --batch
RUN cd site && java -jar bin/gerrit.war reindex
