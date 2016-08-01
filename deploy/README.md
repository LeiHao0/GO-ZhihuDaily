
# How to deploy by supervisor

### Install dependencies (supervisor)

``` bash
    cd {{ project-directory }}
    pip install -r deploy/deps.txt
```

### Edit supervisor program file

Change those fields:
* command : the program (relative uses PATH, can take args)
* user : setuid to this UNIX account to run the program
* directory : directory to cwd to before exec (def no cwd)

``` bash
    cd deploy/programs
    cp zhihudaily-main.conf.sample zhihudaily-main.conf
    vim zhihudaily-main.conf
```

### Supervisor management

``` bash
    # Start
    supervisord -c deploy/supervisord.conf

    # Restart
    supervsiorctl -c deploy/supervisord.conf restart zhihudaily-main

    # Stop
    supervsiorctl -c deploy/supervisord.conf stop zhihudaily-main

    # Show log
    supervsiorctl -c deploy/supervisord.conf tail -f zhihudaily-main
```
