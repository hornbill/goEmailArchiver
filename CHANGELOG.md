# CHANGELOG

## 1.1.0 (September 23rd, 2021)

Changes:
- Addition of "nolocalkeep"-flag to NOT store the email locally - for those who have a backup of their mail server and a local back-up is superfluous.

Fixes:

- removal of email now has correct mailbox connected with it (email ID is unique).

## 1.0.1 (August 18th, 2021)

Fixes:

- There was no check between successful downloads vs expected downloads. Now, if there is a discrepancy, the script will correctly NOT remove the email.
- Addition of "forcedelete"-flag to force removal of email even though download may not have been successful.

## 1.0.0 (August 17th, 2021)

Features:

- Initial Release
