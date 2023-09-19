from string import Template
import subprocess

def parse_yml(path, mapping):
    docker_compose_yaml = ""

    with open(path, "r") as f:
        raw = f.read()
        template = Template(raw)
        docker_compose_yaml = template.safe_substitute(mapping)

    print("docker-compose yml: \r\n{}".format(docker_compose_yaml))

    with open(path, "w") as f:
        f.write(docker_compose_yaml)


def run_docker_compose():
    p = subprocess.Popen(['docker-compose', 'run', '.'], stdout=subprocess.Pipe)
    for line in iter(p.stderr.readline, ""):
        print(line.decode())
    p.wait()

def main():
    parse_yml()

    run_docker_compose()

main()