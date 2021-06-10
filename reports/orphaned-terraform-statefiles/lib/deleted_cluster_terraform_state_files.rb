# List terraform state files in the S3 bucket which belong to non-existent
# clusters
class DeletedClusterTerraformStateFiles
  attr_reader :s3client, :bucket, :cluster_region

  # These prefixes identify terraform statefiles which do not belong to a
  # specific cluster, and which should therefore not be reported as orphaned by
  # this code
  IGNORE_PREFIXES = [
    "cloud-platform-dsd",
    "cloud-platform-environments",
    "cloud-platform-concourse",
    "concourse-pipelines",
    "global-resources",
    "account",
    "terraform.tfstate", # AWS account baseline?
  ]
  IGNORE_SUFFIXES = [
    "account"
  ]

  S3_URL = "https://s3.console.aws.amazon.com/s3/object"

  def initialize(params)
    @s3client = params.fetch(:s3)
    @bucket = params.fetch(:bucket)
    @cluster_region = params.fetch(:cluster_region)
  end

  def list
    exclude_current_clusters(
      exclude_non_cluster_files(
        all_statefiles_in_bucket
      )
    )
  end

  private

  # exclude clusters by fetching the cluster name from the given path
  # This assumes the cluster name pattern as XX/YY/ZZ/<cluster-name>/terraform.tfstate
  def exclude_current_clusters(list)
    list.reject do |hash|
      file = hash.fetch(:file)
      first_part, _, last_part = file.rpartition('/')
      path, _, cluster = first_part.rpartition('/')
      current_clusters.include?(cluster)
    end
  end

  def current_clusters
    @current_clusters ||= ClusterLister.new(region: cluster_region).list
  end

  def exclude_non_cluster_files(list)
    list.reject do |hash|
      file = hash.fetch(:file)
      prefix = file.split("/").first # gets the first folder name of the given path
      first_part, _, last_part = file.rpartition('/') # get the folder path trimming the terraform.tfstate
      suffix = first_part.split("/").last # get the last folder name of the given path
      IGNORE_PREFIXES.include?(prefix) || IGNORE_SUFFIXES.include?(suffix)
    end
  end

  def url(file)
    "#{S3_URL}/#{bucket}?region=#{cluster_region}&prefix=#{file}"
  end

  def all_statefiles_in_bucket
    s3client.bucket(bucket)
      .objects
      .collect(&:key)
      .find_all { |key| key =~ /terraform.tfstate$/ }
      .map { |file| {file: file, url: url(file)} }
  end
end
