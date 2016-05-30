Prerequisites:
==============

 - Experiement description defined in an INI file. See example-exp.ini
   for available options. 

 - [Fabric](http://fabfile.org/) and its Python dependencies available.

 - Your SSH-key available at the remote hosts.

 - `$GOPATH` set locally and the goxos repository available. Path should be
   `$GOPATH/src/github.com/relab/goxos`. 

 - The pitter machines in the unix lab uses the tsch shell as default. Fabric
   executes commands using the bash shell. You may for this reason need to
   ensure that `$GOPATH` is set in your `.bashrc`.


Running:
========

Using the default output path (`/tmp/kvs-experiments`):

```
    $ fab run_exp:example-exp.ini
```

Using a custom output path:
	
```
	$ fab run_exp:example-exp.ini,local_out=/tmp/somedir
```
