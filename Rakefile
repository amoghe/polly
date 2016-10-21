#
# Build the executables
namespace :bin do

  desc "Build the frontend server"
  task :frontman => FileList.new('frontman/*.go') do
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
  task :prod => ["bin:frontman"] do
    container_name = 'polly-prod'
    container_dir  = "container/#{container_name}"
    cp('frontman/frontman', container_dir)

    cmd = build_cmd_for_container(container_name, '1.0')
    Dir.chdir(container_dir) { sh(cmd) }
  end

end

#
# Helpers

# Build a container with the given name (and ver)
def build_cmd_for_container(name, ver="latest")
  dockerfile = "Dockerfile.#{name}"
  return "docker build -t #{name}:#{ver} --file #{dockerfile} ."
end
