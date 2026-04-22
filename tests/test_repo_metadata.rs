use std::fs;
use std::path::Path;

fn read_repo_file(name: &str) -> String {
    let path = Path::new(env!("CARGO_MANIFEST_DIR")).join(name);
    fs::read_to_string(&path).unwrap_or_else(|err| panic!("{name} should be readable: {err}"))
}

#[test]
fn cargo_package_metadata_points_to_laputa() {
    let manifest = read_repo_file("Cargo.toml");

    assert!(
        manifest.contains("repository = \"https://github.com/jxoesneon/laputa\""),
        "Cargo.toml repository must point to Laputa"
    );
    assert!(
        manifest.contains("homepage = \"https://github.com/jxoesneon/laputa\""),
        "Cargo.toml homepage must point to Laputa"
    );
    assert!(
        manifest.contains("documentation = \"https://github.com/jxoesneon/laputa#readme\""),
        "Cargo.toml documentation must point to Laputa's own docs entrypoint"
    );
}

#[test]
fn standalone_docs_do_not_require_sibling_repositories() {
    let readme = read_repo_file("README.md");
    let agents = read_repo_file("AGENTS.md");
    let status = read_repo_file("STATUS.md");

    assert!(
        readme.contains("cargo build"),
        "README must document cargo build as a standalone startup path"
    );
    assert!(
        readme.contains("cargo test"),
        "README must document cargo test as a standalone startup path"
    );
    assert!(
        readme.contains("cargo run -- init"),
        "README must document cargo run -- init as a standalone startup path"
    );

    for forbidden in [
        "../mempalace-rs",
        "../agent-diva",
        "../UPSP",
        "../LifeBook",
    ] {
        assert!(
            !readme.contains(forbidden),
            "README must not require sibling repository path {forbidden}"
        );
        assert!(
            !agents.contains(forbidden),
            "AGENTS must not require sibling repository path {forbidden}"
        );
        assert!(
            !status.contains(forbidden),
            "STATUS must not require sibling repository path {forbidden}"
        );
    }

    assert!(
        status.contains("历史来源") || status.contains("lineage"),
        "STATUS should describe upstream lineage explicitly"
    );
}
