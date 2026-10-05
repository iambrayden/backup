-- Optional: the resource creates this table automatically on start.
CREATE TABLE IF NOT EXISTS `lcrp_halloween_claims` (
    `identifier` VARCHAR(64) NOT NULL,
    `year` SMALLINT UNSIGNED NOT NULL,
    `day` TINYINT UNSIGNED NOT NULL,
    `claimed_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`identifier`, `year`, `day`)
);
