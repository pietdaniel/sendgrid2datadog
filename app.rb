require "dogapi"
require "json"
require "sinatra/base"

class App < Sinatra::Base
  configure do
    set :dog, Dogapi::Client.new(ENV["DD_API_KEY"])
  end

  def dog
    settings.dog
  end

  get "/" do
    "https://github.com/dtan4/sendgrid2datadog"
  end

  post "/webhook" do
    events = JSON.parse(request.body.read)

    events.each do |event|
      dog.emit_point("sendgrid.event.#{event['event']}", 1, timestamp: Time.at(event["timestamp"]), type: "counter")
    end

    "Events was sent to Datadog"
  end
end
