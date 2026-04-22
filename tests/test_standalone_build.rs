use std::fs;
use std::path::Path;

#[test]
fn test_manifest_uses_in_repo_usearch_patch() {
    let manifest_path = Path::new(env!("CARGO_MANIFEST_DIR")).join("Cargo.toml");
    let manifest = fs::read_to_string(&manifest_path).expect("Cargo.toml should be readable");

    // Check usearch patch points to vendor/usearch
    // Flexible matching: allow whitespace variations
    assert!(
        manifest.contains("usearch") && manifest.contains("vendor/usearch"),
        "Cargo.toml must patch usearch from the in-repo vendor directory (vendor/usearch)"
    );

    // Verify the patch section exists with correct path
    assert!(
        manifest.contains("path = \"vendor/usearch\""),
        "Cargo.toml must contain 'path = \"vendor/usearch\"' in patch section"
    );

    // Check no parent-directory path dependencies in path assignments
    // Match pattern: path = "..." where ... contains ../
    for line in manifest.lines() {
        if line.contains("path = \"") && line.contains("../") {
            panic!(
                "Cargo.toml must not contain parent-directory path dependencies: found '{}'",
                line.trim()
            );
        }
    }
}