Feature: Dashboard
  The main dashboard page shows connected boards

  Scenario Outline: Dashboard page loads
    Given I am using a <viewport> viewport
    When I navigate to "/boards"
    Then I should see the heading "My Boards"

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Dashboard shows description
    Given I am using a <viewport> viewport
    When I navigate to "/boards"
    Then I should see "Manage and control your Train PCB boards"

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Board grid container exists
    Given I am using a <viewport> viewport
    When I navigate to "/boards"
    Then the element "#board-grid" should be visible

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Empty state shows setup prompt
    Given I am using a <viewport> viewport
    When I navigate to "/boards"
    Then I should see "No boards yet"

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: HTMX polling configured
    Given I am using a <viewport> viewport
    When I navigate to "/boards"
    Then the element "#board-grid" should have attribute "hx-get" with value "/partials/board-list"
    And the element "#board-grid" should have attribute "hx-trigger" with value "every 5s"

    Examples:
      | viewport |
      | desktop  |
      | mobile   |
