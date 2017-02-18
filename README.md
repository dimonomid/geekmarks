Geekmarks: API-Driven, Geeky Bookmarking Service
================================================================

  * Project page: [Geekmarks](https://geekmarks.dmitryfrank.com/)
  * [Detailed article](https://dmitryfrank.com/projects/geekmarks/article)
  * [Backend API documentation](https://dmitryfrank.com/projects/geekmarks/api)

So I wrote a new bookmarking service. We already have a lot of those, so why
bother writing another one? Good question.

## Rationale

In short, I want my bookmarking service:

  * To be very quick to use;
  * To provide a way to organize my bookmarks in a powerful way.

I tried a lot of existing bookmarking services, and I wasn't satisfied by any
of them, for a variety of reasons.

Let me elaborate on the organization part first. The simplest way to organize
bookmarks is to introduce folders to group them. This still poses a well-known
problem though: some bookmarks can logically belong to multiple folders. In
order to address this issue, some services use tags: now we can tag a bookmark
with more than one tag. So far so good.

Now, assume I have a generic tag programming, and a couple of more specific
tags: python and c. I definitely want my bookmarking service to be smart enough
to figure that if I tag some article with either python or c, it means
programming as well; I don't want to add the tag programming manually every
single time. So, what we need is a hierarchy of tags. Surprisingly enough, I
failed to find a service which would support that.

This hierarchical tags thing was a major motivation for me to start Geekmarks.

Another important thing is that I want bookmarking service to be very quick to
use. I don't want to go through these heavy user interfaces and look at all the
eye candy. In my daily life I just want to either add a bookmark or find one,
and I want to do that quickly: like, just a few keystrokes, and I'm done.

So, meet Geekmarks! A free, open-source,
API-driven bookmarking service.

## Building and running server locally

You'll need [Go](https://golang.org/) 1.6 or higher,
[docker](https://www.docker.com/) and
[docker-compose](https://docs.docker.com/compose/).

You'll also need to create Google OAuth credentials, in order for the
authentication to work (at the moment, authentication is only via Google
account). You can create OAuth credentials in the
[Google Cloud Console](https://console.cloud.google.com/apis/credentials), then
create a file `/var/tmp/geekmarks_dev/main/google_oauth_creds.yaml` with the
following contents:

```
client_id: "your-google-client-id"
client_secret: "your-google-client-secret"
```

Of course, replace placeholders with your actual OAuth credentials.

Make sure you have [`$GOPATH`](https://github.com/golang/go/wiki/GOPATH) set.

Now, clone the repository as `$GOPATH/src/dmitryfrank.com/geekmarks`, and
then from the root of the repo:

```
$ make -C server/envs/dev
```

It will start two containers: posgresql (will be downloaded if needed) and
geekmarks (will be built). Geekmarks backend will listen at the port 4000.

All data will be stored in `/var/tmp/geekmarks_dev/posgresql`.

## Running tests

There are unit tests and integration tests.

For unit tests, only Go is required. For integration tests, docker and
docker-compose are also required.

To run all tests:

```
$ make -C server/envs/test
```

To run unit or integration tests:

```
$ make -C server/envs/test unit-test
$ make -C server/envs/test integration-test
```
