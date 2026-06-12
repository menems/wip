Merge the current feature branch into `main` using rebase + fast-forward.

```
git rebase main
git checkout main
git merge --ff-only feat/<plan-name>
git branch -d feat/<plan-name>
```

- Always rebase first to keep a linear history
- Never use git force flags without user consent
- Always delete the feature branch locally after a successful merge
