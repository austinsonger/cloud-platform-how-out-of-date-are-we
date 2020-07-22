require "spec_helper"

# These specs test a local development server, as started via
#
#     make dev-server
#

HELM_RELEASE_DATA_FILE = "data/helm_whatup.json"

def expect_json_data(url, key)
  response = fetch_url(url, "application/json")
  expect(response.code).to eq("200")
  expect {
    data = JSON.parse(response.body)
    expect(data).to have_key(key)
  }.to_not raise_error
end

describe "local dev server" do
  let(:base_url) { "http://localhost:4567" }
  let(:api_key) { "soopersekrit" } # specified in makefile

  let(:dashboard_url) { [base_url, "dashboard"].join("/") }
  let(:helm_whatup_url) { [base_url, "helm_whatup"].join("/") }
  let(:terraform_modules_url) { [base_url, "terraform_modules"].join("/") }
  let(:documentation_url) { [base_url, "documentation"].join("/") }

  let(:urls) { [
    dashboard_url,
    helm_whatup_url,
    terraform_modules_url,
    documentation_url,
  ] }

  it "redirects / to /dashboard" do
    response = fetch_url(base_url)
    expect(response.code).to eq("302")
    expect(response["location"]).to eq(dashboard_url)
  end

  it "serves pages" do
    urls.each do |url|
      response = fetch_url(url)
      expect(response.code).to eq("200")
    end
  end

  it "serves helm_whatup json" do
    expect_json_data(helm_whatup_url, "clusters")
  end

  it "serves terraform_modules json" do
    expect_json_data(terraform_modules_url, "out_of_date_modules")
  end

  it "serves documentation json" do
    expect_json_data(documentation_url, "pages")
  end

  it "serves dashboard json" do
    expect_json_data(dashboard_url, "data")
  end

  context "with malformed json data" do
    before do
      File.write(HELM_RELEASE_DATA_FILE, " ")
    end

    it "does not crash" do
      response = fetch_url(helm_whatup_url)
      expect(response.code).to eq("200")
    end
  end

  context "updating" do
    context "with no API key" do
      it "rejects" do
        urls.each do |url|
          response = post_to_url(url, "[]")
          expect(response.code).to eq("403")
        end
      end
    end

    context "with correct API key" do
      it "accepts helm_whatup json" do
        json = {
          clusters: [
            name: "live-1",
            apps: []
          ],
          updated_at: Time.now
        }.to_json
        response = post_to_url(helm_whatup_url, json, api_key)
        expect(response.code).to eq("200")
      end

      it "accepts terraform_modules json" do
        json = {
          out_of_date_modules: [],
          updated_at: "2020-04-20",
        }.to_json

        response = post_to_url(terraform_modules_url, json, api_key)
        expect(response.code).to eq("200")
      end

      it "accepts documentation json" do
        json = {
          pages: [],
          updated_at: "2020-04-20",
        }.to_json

        response = post_to_url(documentation_url, json, api_key)
        expect(response.code).to eq("200")
      end
    end

    context "with incorrect API key" do
      let(:api_key) { "this is the wrong API key" }

      it "rejects" do
        urls.each do |url|
          response = post_to_url(url, "[]", api_key)
          expect(response.code).to eq("403")
        end
      end
    end
  end
end
