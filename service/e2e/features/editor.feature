Feature: Plugin Editor
  Create, edit, save, and delete custom LED plugins

  Scenario Outline: Editor page loads
    Given I am using a <viewport> viewport
    When I navigate to "/editor"
    Then the element "#editor-root" should be visible

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: New Plugin button is visible
    Given I am using a <viewport> viewport
    When I navigate to "/editor"
    Then I should see the new plugin button

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Create a new plugin
    Given I am using a <viewport> viewport
    When I navigate to "/editor"
    And I create a new plugin
    Then a plugin should appear in the editor

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Plugin appears in sidebar
    Given I am using a <viewport> viewport
    When I navigate to "/editor"
    And I create a new plugin
    Then the plugin should be listed in the sidebar

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Code editor loads
    Given I am using a <viewport> viewport
    When I navigate to "/editor"
    And I create a new plugin
    Then the code editing area should be visible

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Tab switching to Code
    Given I am using a <viewport> viewport
    When I navigate to "/editor"
    And I create a new plugin
    And I click the tab "Code"
    Then the code editing area should be visible

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Tab switching to Config
    Given I am using a <viewport> viewport
    When I navigate to "/editor"
    And I create a new plugin
    And I click the tab "Config"
    Then I should see "Add Field"

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Tab switching to Info
    Given I am using a <viewport> viewport
    When I navigate to "/editor"
    And I create a new plugin
    And I click the tab "Info"
    Then the element "textarea" should be visible
    And the element "select" should be visible

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Save plugin
    Given I am using a <viewport> viewport
    When I navigate to "/editor"
    And I create a new plugin
    And I save the plugin
    Then the plugin should be saved successfully

    Examples:
      | viewport |
      | desktop  |
      | mobile   |

  Scenario Outline: Delete plugin
    Given I am using a <viewport> viewport
    When I navigate to "/editor"
    And I create a new plugin
    And I delete the plugin
    Then the plugin should be removed from the sidebar

    Examples:
      | viewport |
      | desktop  |
      | mobile   |
