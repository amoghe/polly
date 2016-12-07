FROM polly-base:latest

RUN useradd -s /bin/bash -u 1000 --create-home admin
RUN useradd -s /bin/bash -u 1002 --create-home frontman --groups admin
RUN useradd -s /bin/bash -u 1001 --create-home gerrit   --groups admin

# Copy in the rootfs files, fix the perms
COPY    rootfs/ /
RUN     chown -R admin    /home/admin
RUN     chown -R frontman /home/frontman
RUN     chown -R gerrit   /home/gerrit

# setup admin user (and ssh keys)
USER     admin
WORKDIR  /home/admin
RUN      mkdir .ssh && ssh-keygen -t rsa -N '' -f .ssh/id_rsa

# setup gerrit user (and files)
USER    gerrit
WORKDIR /home/gerrit
RUN     wget https://www.gerritcodereview.com/download/gerrit-2.12.3.war -q

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


USER root
CMD  ["/sbin/my_init"]
