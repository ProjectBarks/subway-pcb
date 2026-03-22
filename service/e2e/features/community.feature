Feature: Community Plugins
  Browse, search, and discover community-created LED plugins

  Scenario Outline: Community page loads
    Given I am using a <viewport> viewport
    When I navigate to "/community"
    Then I should see the heading "Community Plugins"
    And I should see "Browse and install LED patterns created by the community"

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Search input is visible
    Given I am using a <viewport> viewport
    When I navigate to "/community"
    Then the element "input[name='q']" should be visible

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Sort dropdown is visible
    Given I am using a <viewport> viewport
    When I navigate to "/community"
    Then the element "select[name='sort']" should be visible

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Sort dropdown has all options
    Given I am using a <viewport> viewport
    When I navigate to "/community"
    Then the element "select[name='sort']" should contain text "Most Popular"
    And the element "select[name='sort']" should contain text "Recently Updated"
    And the element "select[name='sort']" should contain text "Most Installed"

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Create Plugin link exists
    Given I am using a <viewport> viewport
    When I navigate to "/community"
    Then I should see a link "Create Plugin" to "/editor"

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Plugin grid container exists
    Given I am using a <viewport> viewport
    When I navigate to "/community"
    Then the element "#plugin-grid" should be visible

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Search triggers HTMX update
    Given I am using a <viewport> viewport
    When I navigate to "/community"
    And I search for "test" in the community
    Then the element "#plugin-grid" should be visible

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Sort change triggers update
    Given I am using a <viewport> viewport
    When I navigate to "/community"
    And I change the sort to "Recently Updated"
    Then the element "#plugin-grid" should be visible

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Built-in plugins are displayed
    Given I am using a <viewport> viewport
    When I navigate to "/community"
    Then at least one plugin card should be visible

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Plugin preview canvases render colors
    Given I am using a <viewport> viewport
    When I navigate to "/community"
    Then the canvas previews should render colors

    Examples:
      | viewport |
      | desktop  |
      | mobile   |
