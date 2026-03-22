Feature: Health Check
  The server exposes a health endpoint for monitoring

  Scenario: Health endpoint returns OK
    When I request the health endpoint
    Then the response status should be 200
    And the response should contain "ok"

  Scenario: Health endpoint includes uptime info
    When I request the health endpoint
    Then the response should have uptime_seconds greater than 0
    And the response should have station_count defined
