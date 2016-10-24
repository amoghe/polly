#
# Build the executables
require 'pp'
namespace :bin do

    desc "Build the frontend server"
    task :frontman => FileList.new('./frontman/*.go') do
        Dir.chdir('./frontman') { sh("go build") }
    end

end

#
# Build the containers
namespace :container do

    desc "Build the polly-base container"
    task :base do
        container_name = 'polly-base'
        container_dir  = "container/#{container_name}"

        cmd = build_cmd_for_container(container_name, '1.0')
        Dir.chdir(container_dir) { sh(cmd) }
    end

    desc "Build the polly-prod container"
    task :prod => ['bin:frontman', 'tools:all'] do
        container_name = 'polly-prod'
        container_dir  = "container/#{container_name}"
        cp('frontman/frontman', container_dir)
        cp('tools/create-user/create-user', container_dir)
        cp('tools/create-project/create-project', container_dir)

        cmd = build_cmd_for_container(container_name, '1.0')
        Dir.chdir(container_dir) { sh(cmd) }
    end

end

namespace :tools do

    task :create_user do
        Dir.chdir('tools/create-user') { sh('go build') }
    end

    task :create_project do
        Dir.chdir('tools/create-project') { sh('go build') }
    end

    desc 'Build all the tools'
    task :all => ['create_user', 'create_project']

end

#
# Helpers

# Build a container with the given name (and ver)
def build_cmd_for_container(name, ver="latest")
    dockerfile = "Dockerfile.#{name}"
    return "docker build -t #{name}:#{ver} --file #{dockerfile} ."
end
