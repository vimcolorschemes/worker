class Runner:
    def __init__(self, database):
        self.database = database
        self.last_import_at = self.database.get_last_import_at()

    def store_report(self, job, elapsed_time):
        self.database.create_report({"job": job, "elapsed_time": elapsed_time})
