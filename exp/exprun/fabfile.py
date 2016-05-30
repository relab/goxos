from fabric.api import *
import ConfigParser
import re
import json
import datetime
import time
import sys
import os

env.abort_on_prompts = 'true'
env.colorize_errors = 'true'

env.local_out = '/tmp/kvs-experiments'
env.local_deploy_dir = 'deploy'

env.kvs_script = 'kvs.sh'
env.kvss_script = 'kvss.sh'
env.kvsc_script = 'kvsc.sh'

# Run an experiment
def run_exp(expfile, local_out=None):
    print('-----------------------------')
    print('Starting experiment execution')
    print('-----------------------------\n')

    timestamp, epoch_time = get_timestamp()
    if local_out is not None:
        env.local_out = local_out
    exp_settings = get_settings(expfile)
    set_env(exp_settings)
    check_go_structure()
    create_folders(exp_settings, timestamp)
    checkout_all(exp_settings)
    write_versions(exp_settings)
    build_binaries()
    prepare_state(exp_settings)
    prepare_keyset(exp_settings)
    checkout_head()
    copy_exp_json(exp_settings, expfile)
    write_time(epoch_time)
    generate_configs(expfile)
    generate_scripts(exp_settings)

    deploy()
    for i in range(0, int(exp_settings['runs'])):
        perform_run(i, exp_settings)
    clean_exp()

def get_timestamp():
    return datetime.datetime.now().strftime("%Y%m%d-%H%M%S"), time.time()

def get_settings(expfile):
    print('Loading experiment settings...')
    try:
        with open(expfile) as f:
            cfg = ConfigParser.SafeConfigParser()
            cfg.readfp(f)
            exp_settings = dict(cfg.items("exp"))

            exp_settings['replicas'] =  [i.strip() for i in exp_settings['replicas'].split(",")]
            exp_settings['standbys'] =  [i.strip() for i in exp_settings['standbys'].split(",")]
            exp_settings['clients'] =  [i.strip() for i in exp_settings['clients'].split(",")]

            exp_settings['standbysInt'] =  [i.strip() for i in cfg.get('exp','standbysInt').split(",")]
	    if len(exp_settings['standbysInt']) != len(exp_settings['standbys']):
		exp_settings['standbysInt'] = exp_settings['standbys']

            exp_settings['replicas']= filter(None, exp_settings['replicas'])
            exp_settings['standbys']= filter(None, exp_settings['standbys'])
            exp_settings['clients']= filter(None, exp_settings['clients'])

            return exp_settings
            
    except IOError:
        print('No matching experiment file found. Please ensure that there '
                'is a ini file with path %s' % expfile)
        sys.exit(1)

def set_env(exp_settings):
    print("Setting experiemnt settings to Fabric env...")

    env.user = exp_settings['remoteuser']
    env.remote_scratch = '{0}/{1}'.format(exp_settings['remotescratchdir'], env.user)
    
    env.roledefs = {
        'replica': exp_settings['replicas'],
        'standby': exp_settings['standbys'],
        'client': exp_settings['clients']
    }

    env.local_collect_dir = exp_settings['localcollectdir']
    env.local_ver_out =  exp_settings['localversout']
    env.local_time_out = exp_settings['localtimeout']

    env.glog_name = exp_settings['glogname']
    env.elog_name_replica = exp_settings['elognamereplica']
    env.elog_name_client = exp_settings['elognameclient']
    env.statehash_name = exp_settings['statehashname']

def check_go_structure():
    check_go_path()
    check_go_folders()

def check_go_path():
    print('Checking $GOPATH...')
    gopathset = local('echo $GOPATH', capture=True)
    if not gopathset:
        print('$GOPATH must be set')
        sys.exit(1)

def check_go_folders():
    print('Checking Go folder structure...')
    goxos_src = os.path.expandvars('$GOPATH/src/github.com/relab/goxos')
    if not os.path.isdir(goxos_src):
        print('Need folder $GOPATH/src/github.com/relab/goxos to build binaries')
        sys.exit(1)
    env.goxos_src = goxos_src

def create_folders(exp_settings, timestamp):
    print('Creating local folders...')
    exp_root = os.path.expandvars(env.local_out)
    exp_folder = '{0}-{1}'.format(exp_settings['name'], timestamp)
    env.exp_path = exp_root + '/' + exp_folder
    env.kvs_bin = '{0}/{1}/{2}'.format(env.exp_path, env.local_deploy_dir, 'kvs')
    env.kvsc_bin = '{0}/{1}/{2}'.format(env.exp_path, env.local_deploy_dir, 'kvsc')
    local('mkdir -p {0}'.format(env.exp_path))
    env.local_collect_full_path = '{0}/{1}'.format(env.exp_path, env.local_collect_dir)
    local('mkdir -p {0}'.format(env.local_collect_full_path))

def checkout_all(settings):
    print('Checkout specified git commit for source...')
    with cd(env.goxos_src):
         local('git checkout %s' % settings['goxoscommit'])

def write_versions(exp_settings):
    print('Writing versions...')
    local('go version >> {0}/{1}'.format(env.exp_path, env.local_ver_out))
    local('cd {0} && git rev-parse HEAD >> {1}/{2}'.format(env.goxos_src, env.exp_path, env.local_ver_out))

def build_binaries():
    print('Building needed binaries...')
    with settings(warn_only=True):
            result = local('go build -o {0}/{1} github.com/relab/goxos/kvs/kvsd'.format(env.kvs_bin, 'kvs'))
    if result.failed:
            checkout_head()
            print('kvs build failed!')
            sys.exit(1)
    with settings(warn_only=True):
            result = local('go build -o {0}/{1} github.com/relab/goxos/kvs/kvsc'.format(env.kvsc_bin, 'kvsc'))
    if result.failed:
            checkout_head()
            print('kvsc build failed!')
            sys.exit(1)

def checkout_head():
    print('Reset and checkout HEAD after build...')
    with cd(env.goxos_src):
            local('git checkout HEAD')

def copy_exp_json(exp_settings, expfile):
    print('Copying experiment description...')
    local('cp {0} {1}/{2}'.format(expfile, env.exp_path, exp_settings['localexpdescout']))

def write_time(epoch_time):
    print('Writing timestamp...')
    local('echo {0} > {1}/{2}'.format(epoch_time, env.exp_path, env.local_time_out))

def generate_configs(expfile):
    print('Generating configs...')

    kvs_conf_out = env.kvs_bin + '/' + 'config.ini'
    kvsc_conf_out = env.kvsc_bin + '/' + 'config.ini'

    with open(expfile) as f:
        cfg = ConfigParser.RawConfigParser()
        cfg.optionxform = str
        cfg.readfp(f)

        replicas = [i.strip() for i in cfg.get('exp', 'replicas').split(',')]
        replicasInt = [i.strip() for i in cfg.get('exp', 'replicasInt').split(',')]
        standbys = [i.strip() for i in cfg.get('exp', 'standbys').split(',')]
        standbysInt = [i.strip() for i in cfg.get('exp', 'standbysInt').split(',')]
	if len(replicas) == len(replicasInt) :
	    replicas = replicasInt
	if len(standbys) == len(standbysInt) :
	    standbys = standbysInt

        paxosPort = cfg.get('exp', 'paxosPort')
        clientPort = cfg.get('exp', 'clientPort')
        
        for sec in cfg.sections():
            if sec != "goxos":
                cfg.remove_section(sec)

            nodes = []
            for i, replica in enumerate(replicas):
                nodes.append(str(i) + ":" + replica + ":" + paxosPort + ":" + clientPort)
            nodes = ', '.join(nodes)

            nodeInitStandbys = []
            for standby in standbys:
                nodeInitStandbys.append(standby + ':' + paxosPort + ':' + clientPort)
            nodeInitStandbys = ', '.join(nodeInitStandbys)

        with open(kvs_conf_out, "w") as fh:
            cfg.set("goxos", "nodes", nodes)
            cfg.set("goxos", "nodeInitStandbys", nodeInitStandbys)
            cfg.set("goxos", "nodeInitReplicaProvider", "Kvs")
            cfg.write(fh)

        with open(kvsc_conf_out, "w") as fh:
            cfg.remove_section("goxos")
            cfg.add_section("goxos")
            cfg.set("goxos", "nodes", nodes)
            cfg.write(fh)

def prepare_state(exp_settings):
    print('Preparing initial state deployment...')
    if exp_settings['replicainitialstate']:
        with settings(warn_only=True):
            result = local('cp {0}/{1}/{2} {3}'.format(env.goxosapps_src, 'kvs/state/', exp_settings['replicainitialstate'], env.kvs_bin))
        if result.failed:
            print('Preparing initial state for deployment failed')
            sys.exit(1)

def prepare_keyset(exp_settings):
    print('Preparing key set deployment...')
    if exp_settings['clientkeyset']:
        with settings(warn_only=True):
            result = local('cp {0}/{1}/{2} {3}'.format(env.goxosapps_src, 'kvsc/state', exp_settings['clientkeyset'], env.kvsc_bin))
        if result.failed:
            print('Preparing key set deployment failed')
            sys.exit(1)

def generate_scripts(exp_settings):
    print('Generating remote run commands...')

    rgc_off = str(exp_settings['replicagcoff']).lower()
    init_state = exp_settings['replicainitialstate']
    glog_level = exp_settings['glogvlevel']
    glog_module = exp_settings['glogvmodule']

    # kvs and kvss cmd
    kvs_cmd = './kvs -id $1 -all-cores -gc-off={0} -load-state={1} -log_events -v={2} -vmodule={3} -log_dir=. > /dev/null 2> stderr.txt &'.format(rgc_off, init_state, glog_level, glog_module)
    kvss_cmd = './kvs -mode standby -standby-ip $1 -all-cores -gc-off={0} -log_events -v={1} -vmodule={2} -log_dir=. > /dev/null 2> stderr.txt &'.format(rgc_off, glog_level, glog_module)

    nrc = exp_settings['clientspermachine']
    runs = exp_settings['clientsnumberofruns']
    cmds = exp_settings['clientsnumberofcommands']
    kl = exp_settings['clientskeysize']
    vl = exp_settings['clientsvaluesize']
    keyset = exp_settings['clientkeyset']
    cgc_off = str(exp_settings['clientgcoff']).lower()

    # kvsc cmd
    kvsc_cmd = './kvsc -mode exp -nclients {0} -runs {1} -cmds {2} -kl {3} -vl {4} -keyset="{5}" -gc-off={6} > /dev/null 2> stderr.txt &'.format(nrc, runs, cmds, kl, vl, keyset, cgc_off)

    print('kvs cmd:')
    print(kvs_cmd)
    print('kvss cmd:')
    print(kvss_cmd)
    print('kvsc cmd:')
    print(kvsc_cmd)

    with open(env.kvs_bin + '/' + env.kvs_script, 'w') as text_file:
        text_file.write('#!/bin/sh\n')
        text_file.write(kvs_cmd + '\n')

    with open(env.kvs_bin + '/' + env.kvss_script, 'w') as text_file:
        text_file.write('#!/bin/sh\n')
        text_file.write(kvss_cmd + '\n')

    with open(env.kvsc_bin + '/' + env.kvsc_script, 'w') as text_file:
        text_file.write('#!/bin/sh\n')
        text_file.write(kvsc_cmd + '\n')

    local('chmod +x ' + env.kvs_bin + '/' + env.kvs_script)
    local('chmod +x ' + env.kvs_bin + '/' + env.kvss_script)
    local('chmod +x ' + env.kvsc_bin + '/' + env.kvsc_script)

def deploy():
    print('Deploying...')
    execute(deploy_kvs)
    execute(deploy_kvsc)

@parallel
@roles('replica', 'standby')
def deploy_kvs():
    rdir = env.remote_scratch + '/kvs'
    run('mkdir -p {0}'.format(rdir))
    put(env.kvs_bin, env.remote_scratch, mirror_local_mode=True)

@parallel
@roles('client')
def deploy_kvsc():
    rdir = env.remote_scratch + '/kvsc'
    run('mkdir -p {0}'.format(rdir))
    put(env.kvsc_bin, env.remote_scratch, mirror_local_mode=True)

# Equivalent to golang time.ParseDuration. Returns time in seconds.
def parseDuration(dur):
    units = {
        'ns': 0.000000001,
        'us': 0.000001,
        'ms': 0.001,
        's':  1,
        'm':  60,
        'h':  3600
    }
    result = re.findall('([\\+\\-]?)([\\d]+)\\s*([a-zA-Z]+)\\s*', dur)
    
    total = 0
    for sign, amount, unit in result:
        if sign == '-':
            total -= int(amount) * units[unit]
        else:
            total += int(amount) * units[unit]
    
    return total
    

def perform_run(i, exp_settings):
    print('Performing run number %s...' % i)

    if len(exp_settings['standbys']) > 0:
        print('Starting standbys...')
        execute(run_standbys, exp_settings)

    print('Starting replicas...')
    execute(run_replicas, exp_settings)

    print('Waiting 5 seconds to allow cluster to initialize...')
    time.sleep(5)

    if len(exp_settings['clients']) > 0:
        print('Starting clients...')
        execute(run_clients)

    if not exp_settings['failures']:
        failures = []
    else:
        failures = [{'time': parseDuration(j.split(":")[0].strip()), 'replica': j.split(":")[1]} for j in exp_settings['failures'].split(",")]

    duration = parseDuration(exp_settings['duration'])

    if len(failures) > 0:
        print('Waiting to inject failures...')
        last_failure_time = 0
        for failure in failures:
            time.sleep(failure['time'] - last_failure_time)
            print('Injecting failure: {0}'.format(failure['replica']))
            execute(fail,hosts=failure['replica'])
            last_failure_time = failure['time']
    else:
        print('No failures defined for this experiment...')

    print('Waiting remaining duration of experiment...')
    if len(failures) != 0:
        remaining_sleep = duration - last_failure_time
        if remaining_sleep >= 0:
            time.sleep(remaining_sleep)
    else:
        time.sleep(duration)

    print('Done, experiment duration elapsed.')

    stop_all()
    kill_all() # ensure all are stopped
    collect(i, exp_settings)
    clean_run()

@parallel
@roles('standby')
def run_standbys(exp_settings):
    with cd('{0}/kvs'.format(env.remote_scratch)):
        run('./{0} {1}'.format(env.kvss_script, exp_settings['standbysInt'][exp_settings['standbys'].index(env.host_string)]), pty=False)

@parallel
@roles('replica')
def run_replicas(exp_settings):
    with cd('{0}/kvs'.format(env.remote_scratch)):
        run('./{0} {1}'.format(env.kvs_script, exp_settings['replicas'].index(env.host_string)), pty=False)

@parallel
@roles('client')
def run_clients():
    with cd('{0}/kvsc'.format(env.remote_scratch)):
        run('./{0}'.format(env.kvsc_script), pty=False)

def fail():
    with settings(warn_only=True):
        run('killall kvs')

def stop_all():
    print('Stopping all remote nodes...')
    execute(stop_kvs)
    execute(stop_kvsc)

@parallel
@roles('replica', 'standby')
def stop_kvs():
    print('Stopping replicas and standbys...')
    with settings(warn_only=True):
        run('killall kvs')

@parallel
@roles('client')
def stop_kvsc():
    print('Stopping clients...')
    with settings(warn_only=True):
        run('killall kvsc')

def kill_all():
    print('Killing all remote nodes...')
    execute(kill_kvs)
    execute(kill_kvsc)

@parallel
@roles('replica', 'standby')
def kill_kvs():
    print('Killing replicas and standbys if not already stopped...')
    with settings(warn_only=True):
        run('killall -9 kvs')

@parallel
@roles('client')
def kill_kvsc():
    print('Killing clients if not already stopped...')
    with settings(warn_only=True):
        run('killall -9 kvsc')

def collect(i, exp_settings):
    print('Collecting logs...')
    collect_path = '{0}/{1}'.format(env.local_collect_full_path, i)
    local('mkdir {0}'.format(collect_path))
    execute(collect_kvs, collect_path)
    if len(exp_settings['clients']) > 0:
        execute(collect_kvsc, collect_path)

@parallel
@roles('replica', 'standby')
def collect_kvs(collect_path):
    with settings(warn_only=True):
        get('{0}/kvs/{1}'.format(env.remote_scratch, env.elog_name_replica), '{0}/%(host)s/%(path)s'.format(collect_path))
        get('{0}/kvs/{1}'.format(env.remote_scratch, env.glog_name), '{0}/%(host)s/%(path)s'.format(collect_path))
        # TODO: Don't copy statehash from a failed replica.
        get('{0}/kvs/{1}'.format(env.remote_scratch, env.statehash_name), '{0}/%(host)s/%(path)s'.format(collect_path))
        get('{0}/kvs/stderr.txt'.format(env.remote_scratch), '{0}/%(host)s/%(path)s'.format(collect_path))

@parallel
@roles('client')
def collect_kvsc(collect_path):
    with settings(warn_only=True):
        get('{0}/kvsc/{1}'.format(env.remote_scratch,env.elog_name_client), '{0}/%(host)s/%(path)s'.format(collect_path))
        get('{0}/kvsc/stderr.txt'.format(env.remote_scratch), '{0}/%(host)s/%(path)s'.format(collect_path))

def clean_run():
    print('Cleanup for run...')
    execute(clean_run_kvs)
    execute(clean_run_kvsc)

@parallel
@roles('replica', 'standby')
def clean_run_kvs():
    with settings(warn_only=True):
        run('rm {0}/kvs/*.elog'.format(env.remote_scratch))
        run('rm {0}/kvs/{1}*'.format(env.remote_scratch, env.glog_name))

@parallel
@roles('client')
def clean_run_kvsc():
    with settings(warn_only=True):
        run('rm {0}/kvsc/*.elog'.format(env.remote_scratch))

def clean_exp():
    print('Cleanup for experiment...')
    execute(clean_exp_kvs)
    execute(clean_exp_kvsc)

@parallel
@roles('replica', 'standby')
def clean_exp_kvs():
    with settings(warn_only=True):
        run('rm -r {0}/kvs'.format(env.remote_scratch))

@parallel
@roles('client')
def clean_exp_kvsc():
    with settings(warn_only=True):
        run('rm -r {0}/kvsc'.format(env.remote_scratch))

#######################################################
# Utilities triggered manually when debugging
#######################################################
def clean_man(user):
    env.user = user
    with settings(warn_only=True):
        run('killall -9 kvs')
        run('killall -9 kvsc')
        run('rm -r /local/scratch/{0}/kvs'.format(user))
        run('rm -r /local/scratch/{0}/kvsc'.format(user))

def info(user):
    env.user = user
    with settings(warn_only=True):
        run('w')
        run('free')
        run('ntpstat')


