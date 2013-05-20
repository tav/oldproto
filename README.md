**Getting Started**

First ensure you have the following language runtimes:

* Go
* Java
* NodeJS
* Python
* Ruby

Then install some external tools and libraries with:

```bash
$ sudo easy_install-2.7 -U assetgen
$ sudo easy_install-2.7 -U plumbum
$ sudo easy_install-2.7 -U PyYAML
$ sudo gem install sass
$ sudo npm install -g coffee-script@1.6.2
$ sudo npm install -g uglify-js@2.2.0
$ go get -u github.com/tav/golly
```

And to download App Engine SDKs and other dependencies by running:

```bash
$ ./build install
```

If you have access to the private directory, clone it as the `private`
subdirectory, i.e.

```bash
$ git clone <repo-url> private
```

Symlink the `dist` and `etc` subdirectories:

```bash
$ ln -s private/dist
$ ln -s private/etc
```

This gives you the basic setup. Now, whilst developing you need to run the
following commands concurrently:

```bash
$ ./build app/dev/2
$ ./.appengine_go_sdk/dev_appserver.py app
```

**License**

All of the code has been released into the [Public Domain]. Use it as you
please.

—  
Enjoy, tav <<tav@espians.com>>


[Public Domain]: https://raw.github.com/tav/proto-espra/master/UNLICENSE
