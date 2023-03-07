FROM gcr.io/distroless/base as distroless

FROM scratch

COPY --from=distroless /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ADD ./etcpasswd /etc/passwd
ADD ./bin/mattermost-redmine-bot /var/app/

WORKDIR /var/app
USER nobody

CMD ["/var/app/mattermost-redmine-bot"]
