FROM polly-base:latest

# Ensure users
RUN useradd -s /bin/bash -u 1001 --create-home gerrit
RUN useradd -s /bin/bash -u 1002 --create-home frontman

# Copy in frontman files
USER    frontman
WORKDIR /home/frontman
COPY    rootfs/home/frontman ./

# Copy in gerrit files
USER    gerrit
WORKDIR /home/gerrit
COPY    rootfs/home/gerrit/ ./

# Download gerrit
RUN wget https://www.gerritcodereview.com/download/gerrit-2.12.3.war -q

#
# Initialize gerrit (DEV)
#

RUN cp site/etc/gerrit.config.development site/etc/gerrit.config
RUN java -jar gerrit-2.12.3.war init \
  --site-path site \
  --no-auto-start \
  --batch
RUN cd site && java -jar bin/gerrit.war reindex

# Start gerrit, run the seeds, stop gerrit
RUN cd site && \
  bin/gerrit.sh start && \
  bash seed/do.sh && \
  bin/gerrit.sh stop

#
# Initialize gerrit (PROD)
#

RUN cp site/etc/gerrit.config.production site/etc/gerrit.config
RUN java -jar gerrit-2.12.3.war init \
  --site-path site \
  --no-auto-start \
  --batch
RUN cd site && java -jar bin/gerrit.war reindex

#
# Cleanup
#

# (REMOVEME) Fetch bouncycastle jars
# RUN cd site/lib && wget http://bouncycastle.org/download/bcprov-jdk15on-154.jar -q
# RUN cd site/lib && wget http://bouncycastle.org/download/bcpkix-jdk15on-154.jar -q

# Delete host key
RUN  rm -rf site/etc/ssh_host_key
# Delete seeds
RUN rm -rf seed
