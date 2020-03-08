service(
    "flask",
    image="crccheck/hello-world",
    ports={8000: "localhost:8000"},
    cpu=500,
    memory=256,
)
