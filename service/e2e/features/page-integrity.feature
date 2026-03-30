Feature: Page integrity
  Every page must load without horizontal overflow and with all local static assets returning 200.

  Scenario Outline: No horizontal overflow on <page> (<viewport>)
    Given I am using a <viewport> viewport
    When I navigate to "<path>"
    Then the page should have no horizontal overflow

    Examples:
      | viewport | page      | path       |
      | desktop  | dashboard | /boards    |
      | mobile   | dashboard | /boards    |
      | desktop  | community | /community |
      | mobile   | community | /community |
      | desktop  | editor    | /editor    |
      | mobile   | editor    | /editor    |
      | desktop  | landing   | /landing   |
      | mobile   | landing   | /landing   |

  Scenario Outline: All static assets load on <page> (<viewport>)
    Given I am using a <viewport> viewport
    When I navigate to "<path>"
    Then all local stylesheet and script sources should return 200

    Examples:
      | viewport | page      | path       |
      | desktop  | dashboard | /boards    |
      | mobile   | dashboard | /boards    |
      | desktop  | community | /community |
      | mobile   | community | /community |
      | desktop  | editor    | /editor    |
      | mobile   | editor    | /editor    |
      | desktop  | landing   | /landing   |
      | mobile   | landing   | /landing   |
