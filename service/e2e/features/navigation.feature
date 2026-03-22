Feature: Navigation
  Cross-page navigation via the navigation bar

  Scenario Outline: Root redirects to dashboard
    Given I am using a <viewport> viewport
    When I navigate to "/"
    Then I should be on the dashboard page
    And I should see the heading "My Boards"

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Navigation bar displays logo
    Given I am using a <viewport> viewport
    When I navigate to "/boards"
    Then I should see a link "Train PCB" to "/boards"

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Navigation bar contains all links
    Given I am using a <viewport> viewport
    When I navigate to "/boards"
    Then the page should have nav links to all sections

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Navigate to community via nav bar
    Given I am using a <viewport> viewport
    When I navigate to "/boards"
    And I click "Community" in the navigation bar
    Then I should be on the community page
    And I should see the heading "Community Plugins"

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Navigate to editor via nav bar
    Given I am using a <viewport> viewport
    When I navigate to "/boards"
    And I click "Editor" in the navigation bar
    Then I should be on the editor page
    And the element "#editor-root" should be visible

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Navigate back to dashboard
    Given I am using a <viewport> viewport
    When I navigate to "/community"
    And I click "My Boards" in the navigation bar
    Then I should be on the dashboard page
    And I should see the heading "My Boards"

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Active nav link is highlighted
    Given I am using a <viewport> viewport
    When I navigate to "/boards"
    Then the active nav link should be "My Boards"

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Mobile menu toggle
    Given I am using a <viewport> viewport
    When I navigate to "/boards"
    Then the mobile menu should behave correctly

    Examples:
      | viewport |
      | desktop  |
      | mobile   |
