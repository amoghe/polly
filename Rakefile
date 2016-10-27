#
# Build the executables
namespace :bin do

  file 'frontman/frontman' => FileList['frontman/**/*.go'] do
    sh("cd ./frontman && go build")
  end

  desc "Build the frontend server"
  task :frontman => ['frontman/frontman']

end

#
# Build the containers
namespace :container do

  desc "Build the polly-base container"
  task :base do
    container_name = "polly-base"
    container_dir  = "container/#{container_name}"
    version        = ENV.fetch("VER", "latest")

    Dir.chdir(container_dir) {
      sh("docker build -t #{container_name}:#{version} .")
    }
  end

  desc "Build the polly-prod container"
  task :prod => ["bin:frontman", "tools:all"] do
    container_name = "polly-prod"
    container_dir  = "container/#{container_name}"
    rootfs_dir     = "#{container_dir}/rootfs"
    version        = ENV.fetch("VER", "latest")

    [
      "/home/gerrit",
      "/home/gerrit/site",
      "/home/gerrit/tools",
      "/home/frontman",
    ].each do |dir|
      mkdir_p("#{rootfs_dir}/#{dir}")
    end

    cp('frontman/frontman', "#{rootfs_dir}/home/frontman/")
    cp('tools/create-user/create-user', "#{rootfs_dir}/home/gerrit/tools")
    cp('tools/create-project/create-project', "#{rootfs_dir}/home/gerrit/tools")

    Dir.chdir(container_dir) {
      sh("docker build -t #{container_name}:#{version} .")
    }
  end

  desc "Build the polly-test container"
  task :test do
    container_name = "polly-test"
    container_dir  = "container/#{container_name}"
    version        = ENV.fetch("VER", "latest")

    Dir.chdir(container_dir) {
      sh("docker build -t #{container_name}:#{version} .")
    }
  end

end

#
# tools
namespace :tools do

  def dft(n); "tools/#{n}"; end     # dft: dir for tool
  def eft(n); "#{dft(n)}/#{n}"; end # eft: executable for tool

  file eft("create-user") => FileList["#{dft("create-user")}/*.go"] do
    sh("cd #{dft("create-user")} && go build")
  end

  file eft("create-project") => FileList["#{dft("create-project")}/*.go"] do
    sh("cd #{dft("create-project")} && go build")
  end

  file eft("list-project-acl") => FileList["#{dft("list-project-acl")}/*.go"] do
    sh("cd #{dft("list-project-acl")} && go build")
  end

  file eft("list-groups") => FileList["#{dft("list-groups")}/*.go"] do
    sh("cd #{dft("list-groups")} && go build")
  end

  desc 'Build all the tools'
  task :all => [
                eft("create-user"),
                eft("create-project"),
                eft("list-project-acl"),
                eft("list-groups"),
              ]

end
