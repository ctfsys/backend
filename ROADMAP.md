# Roadmap

This document describes some goals we have for the near future.
The focus of this first round will be devs/ops that will build
it.

Each of these steps will be documented in their own document.
It is encouraged to include links to relevant reading in those
documents.

## 1. Toolchain

Tools and systems that will help us develop this project will be looked at first.
This is everything from build scripts and dev tools, to CI/CD and orchestration software.
This will require a minimal service implementation and an api gateway.

## 2. Conventions and libraries within services

There are a mulitude of options to choose from when we want to do stuff like
talking to a database or validating user input, and more than one of those
options in each case can be suitable. This step is about trying to figure out
how we want to do it, document that, and always do that.

## 3. Fleshing out a single service

We may have tools and conventions that are great by now, but it's very
likely that there are a lot of improvements that could be made.
Developing a sinlge service, and introducing some complexity will
inform revisions of the previous two steps, without buying too much technical debt.
This step is more about gaining experience with our choices in the previous steps,
and less about creating a service we're actually going to use.

## 4. Communication across services

Inter-service communicaion and maintaining state across services is a hard
challenge. In this step we hope to find a method that fits us and ctfsys.

## 5. Contributing

All of the above steps are very code focused. This step is human focused.
We want to create a short guide on how to proceed when you want to file a
bug report, contribute code, and so on. This step should produce guidelines
for both repo owners/admins and contributors.
